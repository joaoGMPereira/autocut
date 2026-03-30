package configurator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func lookFirst(candidates ...string) string {
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}

func statExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// downloadClient is used for all tool downloads.
// A generous timeout prevents indefinite hangs on large binaries.
var downloadClient = &http.Client{
	Timeout: 10 * time.Minute,
}

func downloadFile(ctx context.Context, url, dest string, logCh chan<- string) error {
	logCh <- fmt.Sprintf("Downloading from %s...", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	logCh <- "Installing..."
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	logCh <- "Done!"
	return nil
}

// ---------------------------------------------------------------------------
// YtDlpValidator
// ---------------------------------------------------------------------------

// YtDlpValidator checks and installs yt-dlp (required).
type YtDlpValidator struct {
	dir *AutoCutDir
}

func NewYtDlpValidator(dir *AutoCutDir) *YtDlpValidator {
	return &YtDlpValidator{dir: dir}
}

func (v *YtDlpValidator) Name() string { return "yt-dlp" }

func (v *YtDlpValidator) IsInstalled() bool {
	if _, err := exec.LookPath("yt-dlp"); err == nil {
		return true
	}
	return statExists(v.dir.BinPath("yt-dlp"))
}

func (v *YtDlpValidator) ResolvedPath() string {
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p
	}
	p := v.dir.BinPath("yt-dlp")
	if statExists(p) {
		return p
	}
	return ""
}

func (v *YtDlpValidator) Install(ctx context.Context, logCh chan<- string) error {
	logCh <- "Downloading yt-dlp..."

	var filename string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		filename = "yt-dlp_macos"
	case runtime.GOOS == "darwin":
		filename = "yt-dlp_macos_legacy"
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		filename = "yt-dlp_linux_aarch64"
	case runtime.GOOS == "linux":
		filename = "yt-dlp_linux"
	case runtime.GOOS == "windows":
		filename = "yt-dlp.exe"
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	url := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/" + filename
	dest := v.dir.BinPath("yt-dlp")

	if err := downloadFile(ctx, url, dest, logCh); err != nil {
		return err
	}

	slog.Info("yt-dlp installed", "component", "configurator", "path", dest)
	return nil
}

func (v *YtDlpValidator) Instructions() string {
	return "https://github.com/yt-dlp/yt-dlp#installation"
}

func (v *YtDlpValidator) Status() ToolStatus {
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Required:  true,
	}
}

// ---------------------------------------------------------------------------
// TwitchCLIValidator
// ---------------------------------------------------------------------------

const twitchVersion = "1.54.5"

// TwitchCLIValidator checks and installs TwitchDownloaderCLI (optional).
type TwitchCLIValidator struct {
	dir *AutoCutDir
}

func NewTwitchCLIValidator(dir *AutoCutDir) *TwitchCLIValidator {
	return &TwitchCLIValidator{dir: dir}
}

func (v *TwitchCLIValidator) Name() string { return "TwitchDownloaderCLI" }

func (v *TwitchCLIValidator) IsInstalled() bool {
	if _, err := exec.LookPath("TwitchDownloaderCLI"); err == nil {
		return true
	}
	candidates := []string{
		"/usr/local/bin/TwitchDownloaderCLI",
		"/usr/bin/TwitchDownloaderCLI",
		v.dir.BinPath("TwitchDownloaderCLI"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", "TwitchDownloaderCLI"))
	}
	for _, c := range candidates {
		if statExists(c) {
			return true
		}
	}
	return false
}

func (v *TwitchCLIValidator) ResolvedPath() string {
	if p, err := exec.LookPath("TwitchDownloaderCLI"); err == nil {
		return p
	}
	candidates := []string{
		"/usr/local/bin/TwitchDownloaderCLI",
		"/usr/bin/TwitchDownloaderCLI",
		v.dir.BinPath("TwitchDownloaderCLI"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", "TwitchDownloaderCLI"))
	}
	for _, c := range candidates {
		if statExists(c) {
			return c
		}
	}
	return ""
}

func (v *TwitchCLIValidator) Install(ctx context.Context, logCh chan<- string) error {
	logCh <- "Downloading TwitchDownloaderCLI..."

	var filename string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		filename = fmt.Sprintf("TwitchDownloaderCLI-%s-osx-arm64", twitchVersion)
	case runtime.GOOS == "darwin":
		filename = fmt.Sprintf("TwitchDownloaderCLI-%s-osx-x64", twitchVersion)
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		filename = fmt.Sprintf("TwitchDownloaderCLI-%s-Linux-arm64", twitchVersion)
	case runtime.GOOS == "linux":
		filename = fmt.Sprintf("TwitchDownloaderCLI-%s-Linux-x64", twitchVersion)
	case runtime.GOOS == "windows":
		filename = "TwitchDownloaderCLI.exe"
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	url := "https://github.com/lay295/TwitchDownloader/releases/latest/download/" + filename
	dest := v.dir.BinPath("TwitchDownloaderCLI")

	if err := downloadFile(ctx, url, dest, logCh); err != nil {
		return err
	}

	slog.Info("TwitchDownloaderCLI installed", "component", "configurator", "path", dest)
	return nil
}

func (v *TwitchCLIValidator) Instructions() string {
	return "https://github.com/lay295/TwitchDownloader/releases"
}

func (v *TwitchCLIValidator) Status() ToolStatus {
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Required:  false,
	}
}

// ---------------------------------------------------------------------------
// FFmpegValidator
// ---------------------------------------------------------------------------

// FFmpegValidator checks for FFmpeg (required, no auto-install).
type FFmpegValidator struct {
	dir *AutoCutDir
}

func NewFFmpegValidator(dir *AutoCutDir) *FFmpegValidator {
	return &FFmpegValidator{dir: dir}
}

func (v *FFmpegValidator) Name() string { return "ffmpeg" }

func (v *FFmpegValidator) IsInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func (v *FFmpegValidator) ResolvedPath() string {
	p, err := exec.LookPath("ffmpeg")
	if err != nil {
		return ""
	}
	return p
}

func (v *FFmpegValidator) Install(_ context.Context, _ chan<- string) error {
	return fmt.Errorf("ffmpeg requires manual installation: %s", v.Instructions())
}

func (v *FFmpegValidator) Instructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install ffmpeg"
	case "linux":
		return "apt install ffmpeg"
	case "windows":
		return "winget install ffmpeg"
	default:
		return "https://ffmpeg.org/download.html"
	}
}

func (v *FFmpegValidator) Status() ToolStatus {
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Required:  true,
	}
}

// ---------------------------------------------------------------------------
// WhisperValidator
// ---------------------------------------------------------------------------

// WhisperValidator checks for whisper / whisper-cli (optional).
type WhisperValidator struct {
	dir *AutoCutDir
}

func NewWhisperValidator(dir *AutoCutDir) *WhisperValidator {
	return &WhisperValidator{dir: dir}
}

func (v *WhisperValidator) Name() string { return "whisper" }

func (v *WhisperValidator) IsInstalled() bool {
	return v.ResolvedPath() != ""
}

func (v *WhisperValidator) ResolvedPath() string {
	// Priority: whisper-cli in PATH, whisper in PATH, then well-known paths
	if p := lookFirst("whisper-cli", "whisper"); p != "" {
		return p
	}
	candidates := []string{
		v.dir.BinPath("whisper"),
		"/usr/local/bin/whisper-cli",
	}
	for _, c := range candidates {
		if statExists(c) {
			return c
		}
	}
	return ""
}

func (v *WhisperValidator) Install(_ context.Context, logCh chan<- string) error {
	logCh <- "Manual build required: see https://github.com/ggerganov/whisper.cpp"
	return nil
}

func (v *WhisperValidator) Instructions() string {
	return "brew install whisper.cpp (macOS) or build from source: https://github.com/ggerganov/whisper.cpp"
}

func (v *WhisperValidator) Status() ToolStatus {
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Required:  false,
	}
}

// ---------------------------------------------------------------------------
// OllamaValidator
// ---------------------------------------------------------------------------

// OllamaValidator checks for ollama binary AND a live server (optional).
type OllamaValidator struct {
	dir        *AutoCutDir
	healthURL  string // injectable for tests; default "http://localhost:11434/"
	httpClient *http.Client
}

func NewOllamaValidator(dir *AutoCutDir) *OllamaValidator {
	return &OllamaValidator{
		dir:       dir,
		healthURL: "http://localhost:11434/",
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// newOllamaValidatorWithURL creates a validator that uses a custom health URL.
// Used in tests to point at a port that is guaranteed to be closed.
func newOllamaValidatorWithURL(dir *AutoCutDir, url string) *OllamaValidator {
	v := NewOllamaValidator(dir)
	v.healthURL = url
	return v
}

func (v *OllamaValidator) Name() string { return "ollama" }

func (v *OllamaValidator) IsInstalled() bool {
	if _, err := exec.LookPath("ollama"); err != nil {
		return false
	}
	resp, err := v.httpClient.Get(v.healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

func (v *OllamaValidator) ResolvedPath() string {
	p, err := exec.LookPath("ollama")
	if err != nil {
		return ""
	}
	return p
}

func (v *OllamaValidator) Install(_ context.Context, _ chan<- string) error {
	return fmt.Errorf("ollama requires manual installation: %s", v.Instructions())
}

func (v *OllamaValidator) Instructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install ollama"
	default:
		return "https://ollama.ai/download"
	}
}

func (v *OllamaValidator) Status() ToolStatus {
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Required:  false,
	}
}

// ---------------------------------------------------------------------------
// ImageMagickValidator
// ---------------------------------------------------------------------------

// ImageMagickValidator checks for ImageMagick (optional).
type ImageMagickValidator struct {
	dir *AutoCutDir
}

func NewImageMagickValidator(dir *AutoCutDir) *ImageMagickValidator {
	return &ImageMagickValidator{dir: dir}
}

func (v *ImageMagickValidator) Name() string { return "convert" }

func (v *ImageMagickValidator) IsInstalled() bool {
	return v.ResolvedPath() != ""
}

func (v *ImageMagickValidator) ResolvedPath() string {
	if p, err := exec.LookPath("convert"); err == nil {
		return p
	}
	if p, err := exec.LookPath("magick"); err == nil {
		return p
	}
	return ""
}

func (v *ImageMagickValidator) Install(_ context.Context, _ chan<- string) error {
	return fmt.Errorf("ImageMagick requires manual installation: %s", v.Instructions())
}

func (v *ImageMagickValidator) Instructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install imagemagick"
	case "linux":
		return "apt install imagemagick"
	default:
		return "https://imagemagick.org/script/download.php"
	}
}

func (v *ImageMagickValidator) Status() ToolStatus {
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Required:  false,
	}
}
