package configurator

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// discoverLocalBinary searches well-known system paths for binName without
// relying on the PATH environment variable. This allows AutoCut to find tools
// installed via Homebrew or system packages even in restricted environments.
func discoverLocalBinary(binName string) string {
	dirs := []string{
		"/opt/homebrew/bin",  // macOS Apple Silicon Homebrew
		"/opt/homebrew/sbin",
		"/usr/local/bin", // macOS Intel Homebrew / Linux manual installs
		"/usr/local/sbin",
		"/usr/bin", // Linux system packages
		"/usr/sbin",
		"/bin",
	}
	for _, dir := range dirs {
		if p := filepath.Join(dir, binName); statExists(p) {
			return p
		}
	}
	return ""
}

// downloadToTemp downloads url to a temporary file and returns its path.
// The caller is responsible for removing the temp file after use.
func downloadToTemp(ctx context.Context, url string, logCh chan<- string) (string, error) {
	logCh <- fmt.Sprintf("Downloading from %s...", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "autocut-download-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}
	tmp.Close()
	return tmp.Name(), nil
}

// extractFromZip extracts the first entry whose base name equals binPattern
// from the zip archive at zipPath, writing it to destPath with mode 0755.
func extractFromZip(zipPath, binPattern, destPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) != binPattern {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		err = writeFromReader(rc, destPath)
		rc.Close()
		return err
	}
	return fmt.Errorf("%q not found in zip archive", binPattern)
}

// extractFromTarGz extracts the first entry whose base name equals binPattern
// from the tar.gz archive at tarPath, writing it to destPath with mode 0755.
func extractFromTarGz(tarPath, binPattern, destPath string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != binPattern {
			continue
		}
		return writeFromReader(tr, destPath)
	}
	return fmt.Errorf("%q not found in tar.gz archive", binPattern)
}

// writeFromReader writes r to destPath atomically with mode 0755.
func writeFromReader(r io.Reader, destPath string) error {
	tmp := destPath + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	if _, err := io.Copy(out, r); err != nil {
		out.Close()
		os.Remove(tmp)
		return fmt.Errorf("write: %w", err)
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close: %w", err)
	}
	return os.Rename(tmp, destPath)
}

// ---------------------------------------------------------------------------
// ffmpeg download — platform-specific helpers
// ---------------------------------------------------------------------------

// evermeetResponse is the JSON structure returned by the evermeet.cx release API.
type evermeetResponse struct {
	Download string `json:"download"`
}

// fetchGitHubReleaseAsset queries the GitHub Releases API for the latest release
// of owner/repo and returns the browser_download_url of the first asset whose
// name ends with assetSuffix. This avoids hardcoding version numbers.
func fetchGitHubReleaseAsset(ctx context.Context, owner, repo, assetSuffix string, logCh chan<- string) (string, error) {
	logCh <- fmt.Sprintf("Fetching latest %s/%s release info...", owner, repo)

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "AutoCut-App")

	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch release info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d for %s/%s", resp.StatusCode, owner, repo)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode release info: %w", err)
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, assetSuffix) {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no asset with suffix %q found in latest %s/%s release", assetSuffix, owner, repo)
}

// fetchEvermeetFFmpeg downloads the latest static macOS ffmpeg build from
// evermeet.cx and installs it at destPath.
func fetchEvermeetFFmpeg(ctx context.Context, destPath string, logCh chan<- string) error {
	logCh <- "Fetching ffmpeg release info from evermeet.cx..."

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://evermeet.cx/ffmpeg/getrelease/zip", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch release info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("evermeet.cx returned status %d", resp.StatusCode)
	}

	var info evermeetResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return fmt.Errorf("decode release info: %w", err)
	}
	if info.Download == "" {
		return fmt.Errorf("evermeet.cx returned empty download URL")
	}

	tmp, err := downloadToTemp(ctx, info.Download, logCh)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)

	logCh <- "Extracting ffmpeg..."
	return extractFromZip(tmp, "ffmpeg", destPath)
}

// downloadFFmpegLinux downloads the latest static ffmpeg build for Linux amd64
// from BtbN and extracts it using the system tar command (avoids xz dependency).
func downloadFFmpegLinux(ctx context.Context, dest string, logCh chan<- string) error {
	url := "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz"
	tmp, err := downloadToTemp(ctx, url, logCh)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)

	logCh <- "Extracting ffmpeg..."
	tmpDir, err := os.MkdirTemp("", "autocut-ffmpeg-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// System tar with xz support (-J) is available on all modern Linux distributions.
	cmd := exec.CommandContext(ctx, "tar", "-xJf", tmp,
		"--wildcards", "*/bin/ffmpeg",
		"-C", tmpDir,
		"--strip-components=2",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("extract tar.xz: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return copyFile(filepath.Join(tmpDir, "ffmpeg"), dest)
}

// downloadFFmpegWindows downloads the latest static ffmpeg build for Windows
// from BtbN and extracts ffmpeg.exe from the zip.
func downloadFFmpegWindows(ctx context.Context, dest string, logCh chan<- string) error {
	url := "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"
	tmp, err := downloadToTemp(ctx, url, logCh)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)

	logCh <- "Extracting ffmpeg.exe..."
	return extractFromZip(tmp, "ffmpeg.exe", dest)
}

// ---------------------------------------------------------------------------
// Homebrew helper
// ---------------------------------------------------------------------------

// chanWriter adapts a chan<- string to io.Writer so subprocess stdout/stderr
// can be streamed line-by-line to the SSE log channel.
type chanWriter struct {
	ch chan<- string
}

func (w *chanWriter) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if line != "" {
			w.ch <- line
		}
	}
	return len(p), nil
}

// discoverLocalDylib returns the path to libwhisper.1.dylib on the system, or empty string.
func discoverLocalDylib() string {
	candidates := []string{
		"/opt/homebrew/lib/libwhisper.1.dylib",
		"/usr/local/lib/libwhisper.1.dylib",
	}
	for _, p := range candidates {
		if statExists(p) {
			return p
		}
	}
	return ""
}

// copyToLibWithProgress copies a file from src into dir.LibDir/filename, logging progress.
func copyToLibWithProgress(dir *AutoCutDir, filename, src string, logCh chan<- string) error {
	dst := dir.LibPath(filename)
	logCh <- fmt.Sprintf("Copying %s → %s", src, dst)
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("copy %s: %w", filename, err)
	}
	logCh <- fmt.Sprintf("Installed %s", filename)
	return nil
}

// installViaBrewAndCopy runs `brew install formula`, streams its output to
// logCh, then searches binCandidates in PATH and well-known paths and copies
// the first match to dir.BinPath(binName).
func installViaBrewAndCopy(ctx context.Context, brewPath, formula string, dir *AutoCutDir, binName string, binCandidates []string, logCh chan<- string) error {
	logCh <- fmt.Sprintf("Installing %s via Homebrew (this may take a moment)...", formula)

	cmd := exec.CommandContext(ctx, brewPath, "install", formula)
	cmd.Stdout = &chanWriter{logCh}
	cmd.Stderr = &chanWriter{logCh}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew install %s: %w", formula, err)
	}

	// After brew finishes, search for the installed binary.
	for _, name := range binCandidates {
		if found := discoverLocalBinary(name); found != "" {
			return copyToBinWithProgress(dir, binName, found, logCh)
		}
	}
	for _, name := range binCandidates {
		if found, err := exec.LookPath(name); err == nil {
			return copyToBinWithProgress(dir, binName, found, logCh)
		}
	}
	return fmt.Errorf("brew install succeeded but binary not found — try restarting the app")
}
