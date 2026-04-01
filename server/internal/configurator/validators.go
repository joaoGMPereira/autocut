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
	"strings"
	"time"
)

// runVersionCmd executes a command with a 3-second timeout and returns trimmed stdout.
// Returns "" on any error.
func runVersionCmd(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

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

// copyFile copies src → dst preserving executable permissions (0755).
// Uses a temp-then-rename strategy so the destination is never partially written.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return fmt.Errorf("copy data: %w", err)
	}
	if err = out.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close dest: %w", err)
	}
	return os.Rename(tmp, dst)
}

// autoCopyBackground copies systemPath → ~/.autocut/bin/binName in a background
// goroutine if the destination does not already exist. Idempotent and
// best-effort — errors are silenced so a failed copy never breaks status checks.
func autoCopyBackground(dir *AutoCutDir, binName, systemPath string) {
	if systemPath == "" || statExists(dir.BinPath(binName)) {
		return
	}
	go func() {
		_ = copyFile(systemPath, dir.BinPath(binName))
	}()
}

// copyToBinWithProgress copies systemPath → ~/.autocut/bin/binName, logging
// progress to logCh. Returns error if systemPath is not found or copy fails.
func copyToBinWithProgress(dir *AutoCutDir, binName, systemPath string, logCh chan<- string) error {
	dest := dir.BinPath(binName)
	if statExists(dest) {
		logCh <- binName + " is already in ~/.autocut/bin/, skipping copy."
		return nil
	}
	logCh <- fmt.Sprintf("Copying %s from %s...", binName, systemPath)
	if err := copyFile(systemPath, dest); err != nil {
		return fmt.Errorf("copy %s: %w", binName, err)
	}
	logCh <- fmt.Sprintf("%s installed to %s", binName, dest)
	return nil
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
	if statExists(v.dir.BinPath("yt-dlp")) {
		return true
	}
	_, err := exec.LookPath("yt-dlp")
	return err == nil
}

func (v *YtDlpValidator) ResolvedPath() string {
	if p := v.dir.BinPath("yt-dlp"); statExists(p) {
		return p
	}
	if p, err := exec.LookPath("yt-dlp"); err == nil {
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

func (v *YtDlpValidator) Version() string {
	p := v.ResolvedPath()
	if p == "" {
		return ""
	}
	return runVersionCmd(p, "--version")
}

func (v *YtDlpValidator) Status() ToolStatus {
	if systemPath, err := exec.LookPath("yt-dlp"); err == nil {
		autoCopyBackground(v.dir, "yt-dlp", systemPath)
	}
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Version:   v.Version(),
		Required:  true,
	}
}

// ---------------------------------------------------------------------------
// TwitchCLIValidator
// ---------------------------------------------------------------------------

// TwitchCLIValidator checks and installs TwitchDownloaderCLI (optional).
type TwitchCLIValidator struct {
	dir *AutoCutDir
}

func NewTwitchCLIValidator(dir *AutoCutDir) *TwitchCLIValidator {
	return &TwitchCLIValidator{dir: dir}
}

func (v *TwitchCLIValidator) Name() string { return "TwitchDownloaderCLI" }

func (v *TwitchCLIValidator) IsInstalled() bool {
	if statExists(v.dir.BinPath("TwitchDownloaderCLI")) {
		return true
	}
	if _, err := exec.LookPath("TwitchDownloaderCLI"); err == nil {
		return true
	}
	candidates := []string{
		"/usr/local/bin/TwitchDownloaderCLI",
		"/usr/bin/TwitchDownloaderCLI",
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
	if p := v.dir.BinPath("TwitchDownloaderCLI"); statExists(p) {
		return p
	}
	if p, err := exec.LookPath("TwitchDownloaderCLI"); err == nil {
		return p
	}
	candidates := []string{
		"/usr/local/bin/TwitchDownloaderCLI",
		"/usr/bin/TwitchDownloaderCLI",
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
	// Asset names as of v1.56+: TwitchDownloaderCLI-<ver>-<suffix>.zip
	// Queried from the GitHub Releases API to avoid hardcoding versions.
	var assetSuffix, binInZip string
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		assetSuffix, binInZip = "MacOSArm64.zip", "TwitchDownloaderCLI"
	case runtime.GOOS == "darwin":
		assetSuffix, binInZip = "MacOS-x64.zip", "TwitchDownloaderCLI"
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		assetSuffix, binInZip = "LinuxArm64.zip", "TwitchDownloaderCLI"
	case runtime.GOOS == "linux":
		assetSuffix, binInZip = "Linux-x64.zip", "TwitchDownloaderCLI"
	case runtime.GOOS == "windows":
		assetSuffix, binInZip = "Windows-x64.zip", "TwitchDownloaderCLI.exe"
	default:
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	downloadURL, err := fetchGitHubReleaseAsset(ctx, "lay295", "TwitchDownloader", assetSuffix, logCh)
	if err != nil {
		return err
	}

	tmp, err := downloadToTemp(ctx, downloadURL, logCh)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)

	logCh <- "Extracting TwitchDownloaderCLI..."
	dest := v.dir.BinPath("TwitchDownloaderCLI")
	if err := extractFromZip(tmp, binInZip, dest); err != nil {
		return err
	}

	slog.Info("TwitchDownloaderCLI installed", "component", "configurator", "path", dest)
	return nil
}

func (v *TwitchCLIValidator) Instructions() string {
	return "https://github.com/lay295/TwitchDownloader/releases"
}

func (v *TwitchCLIValidator) Version() string {
	p := v.ResolvedPath()
	if p == "" {
		return ""
	}
	out := runVersionCmd(p, "--version")
	if line, _, ok := strings.Cut(out, "\n"); ok {
		return strings.TrimSpace(line)
	}
	return out
}

func (v *TwitchCLIValidator) Status() ToolStatus {
	if systemPath, err := exec.LookPath("TwitchDownloaderCLI"); err == nil {
		autoCopyBackground(v.dir, "TwitchDownloaderCLI", systemPath)
	}
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Version:   v.Version(),
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
	if statExists(v.dir.BinPath("ffmpeg")) {
		return true
	}
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func (v *FFmpegValidator) ResolvedPath() string {
	if p := v.dir.BinPath("ffmpeg"); statExists(p) {
		return p
	}
	p, err := exec.LookPath("ffmpeg")
	if err != nil {
		return ""
	}
	return p
}

func (v *FFmpegValidator) Install(ctx context.Context, logCh chan<- string) error {
	if statExists(v.dir.BinPath("ffmpeg")) {
		logCh <- "ffmpeg already in ~/.autocut/bin/"
		return nil
	}
	// Fast path: copy from a known local installation (no download needed).
	if found := discoverLocalBinary("ffmpeg"); found != "" {
		logCh <- fmt.Sprintf("Found ffmpeg at %s, copying...", found)
		return copyToBinWithProgress(v.dir, "ffmpeg", found, logCh)
	}
	// Also try PATH in case it's in a non-standard location.
	if systemPath, err := exec.LookPath("ffmpeg"); err == nil {
		logCh <- fmt.Sprintf("Found ffmpeg at %s, copying...", systemPath)
		return copyToBinWithProgress(v.dir, "ffmpeg", systemPath, logCh)
	}
	// Download from the internet.
	dest := v.dir.BinPath("ffmpeg")
	switch runtime.GOOS {
	case "darwin":
		return fetchEvermeetFFmpeg(ctx, dest, logCh)
	case "linux":
		return downloadFFmpegLinux(ctx, dest, logCh)
	case "windows":
		return downloadFFmpegWindows(ctx, dest, logCh)
	default:
		return fmt.Errorf("unsupported platform — install manually: %s", v.Instructions())
	}
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

func (v *FFmpegValidator) Version() string {
	p := v.ResolvedPath()
	if p == "" {
		return ""
	}
	out := runVersionCmd(p, "-version")
	line, _, _ := strings.Cut(out, "\n")
	// "ffmpeg version 6.1 ..."
	if _, after, ok := strings.Cut(line, "ffmpeg version "); ok {
		ver, _, _ := strings.Cut(after, " ")
		return ver
	}
	return ""
}

func (v *FFmpegValidator) Status() ToolStatus {
	if systemPath, err := exec.LookPath("ffmpeg"); err == nil {
		autoCopyBackground(v.dir, "ffmpeg", systemPath)
	}
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Version:   v.Version(),
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
	// Priority: ~/.autocut/bin/ first, then PATH, then well-known paths
	for _, binName := range []string{"whisper-cli", "whisper"} {
		if p := v.dir.BinPath(binName); statExists(p) {
			return p
		}
	}
	if p := lookFirst("whisper-cli", "whisper"); p != "" {
		return p
	}
	if statExists("/usr/local/bin/whisper-cli") {
		return "/usr/local/bin/whisper-cli"
	}
	return ""
}

func (v *WhisperValidator) Install(ctx context.Context, logCh chan<- string) error {
	if statExists(v.dir.BinPath("whisper-cli")) {
		logCh <- "whisper already in ~/.autocut/bin/"
		return nil
	}
	// Try PATH first.
	if systemPath := lookFirst("whisper-cli", "whisper", "whisper-cpp"); systemPath != "" {
		return copyToBinWithProgress(v.dir, "whisper-cli", systemPath, logCh)
	}
	// Try well-known system paths (Homebrew, /usr/local/bin, etc.).
	for _, name := range []string{"whisper-cli", "whisper", "whisper-cpp"} {
		if found := discoverLocalBinary(name); found != "" {
			return copyToBinWithProgress(v.dir, "whisper-cli", found, logCh)
		}
	}
	// macOS: whisper.cpp has no pre-built GitHub release binary; use Homebrew.
	if runtime.GOOS == "darwin" {
		if brewPath, err := exec.LookPath("brew"); err == nil {
			return installViaBrewAndCopy(ctx, brewPath, "whisper-cpp", v.dir, "whisper-cli",
				[]string{"whisper-cli", "whisper", "whisper-cpp"}, logCh)
		}
	}
	return fmt.Errorf("whisper not found — install it first: %s", v.Instructions())
}

func (v *WhisperValidator) Instructions() string {
	return "brew install whisper-cpp (macOS) or build from source: https://github.com/ggerganov/whisper.cpp"
}

func (v *WhisperValidator) Version() string {
	p := v.ResolvedPath()
	if p == "" {
		return ""
	}
	out := runVersionCmd(p, "--version")
	if out == "" {
		return ""
	}
	line, _, _ := strings.Cut(out, "\n")
	return strings.TrimSpace(line)
}

func (v *WhisperValidator) Status() ToolStatus {
	// Auto-symlink: prefer whisper-cli name as the canonical bin/ entry
	if systemPath := lookFirst("whisper-cli", "whisper"); systemPath != "" {
		autoCopyBackground(v.dir, "whisper-cli", systemPath)
	}
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Version:   v.Version(),
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
	// Binary check: ~/.autocut/bin/ first, then PATH
	hasBinary := statExists(v.dir.BinPath("ollama"))
	if !hasBinary {
		_, err := exec.LookPath("ollama")
		hasBinary = err == nil
	}
	if !hasBinary {
		return false
	}
	// Server must also be running
	resp, err := v.httpClient.Get(v.healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

func (v *OllamaValidator) ResolvedPath() string {
	if p := v.dir.BinPath("ollama"); statExists(p) {
		return p
	}
	p, err := exec.LookPath("ollama")
	if err != nil {
		return ""
	}
	return p
}

func (v *OllamaValidator) Install(ctx context.Context, logCh chan<- string) error {
	if statExists(v.dir.BinPath("ollama")) {
		logCh <- "ollama already in ~/.autocut/bin/"
		return nil
	}
	// Fast path: copy from a known local installation.
	if found := discoverLocalBinary("ollama"); found != "" {
		logCh <- fmt.Sprintf("Found ollama at %s, copying...", found)
		return copyToBinWithProgress(v.dir, "ollama", found, logCh)
	}
	if systemPath, err := exec.LookPath("ollama"); err == nil {
		logCh <- fmt.Sprintf("Found ollama at %s, copying...", systemPath)
		return copyToBinWithProgress(v.dir, "ollama", systemPath, logCh)
	}
	// Download from the internet.
	switch runtime.GOOS {
	case "darwin":
		url := "https://github.com/ollama/ollama/releases/latest/download/ollama-darwin.tgz"
		tmp, err := downloadToTemp(ctx, url, logCh)
		if err != nil {
			return err
		}
		defer os.Remove(tmp)
		logCh <- "Extracting ollama..."
		return extractFromTarGz(tmp, "ollama", v.dir.BinPath("ollama"))
	default:
		return fmt.Errorf("automatic install not supported on %s — install manually: %s", runtime.GOOS, v.Instructions())
	}
}

func (v *OllamaValidator) Instructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install ollama"
	default:
		return "https://ollama.ai/download"
	}
}

func (v *OllamaValidator) Version() string {
	p := v.ResolvedPath()
	if p == "" {
		return ""
	}
	out := runVersionCmd(p, "--version")
	// "ollama version is 0.1.32" → last word
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func (v *OllamaValidator) Status() ToolStatus {
	if systemPath, err := exec.LookPath("ollama"); err == nil {
		autoCopyBackground(v.dir, "ollama", systemPath)
	}
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Version:   v.Version(),
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
	if p := v.dir.BinPath("convert"); statExists(p) {
		return p
	}
	if p, err := exec.LookPath("convert"); err == nil {
		return p
	}
	if p, err := exec.LookPath("magick"); err == nil {
		return p
	}
	return ""
}

func (v *ImageMagickValidator) Install(ctx context.Context, logCh chan<- string) error {
	if statExists(v.dir.BinPath("convert")) {
		logCh <- "ImageMagick already in ~/.autocut/bin/"
		return nil
	}
	// Try PATH first.
	if systemPath := lookFirst("convert", "magick"); systemPath != "" {
		return copyToBinWithProgress(v.dir, "convert", systemPath, logCh)
	}
	// Try well-known system paths.
	for _, name := range []string{"convert", "magick"} {
		if found := discoverLocalBinary(name); found != "" {
			return copyToBinWithProgress(v.dir, "convert", found, logCh)
		}
	}
	// macOS: ImageMagick has no suitable standalone binary; use Homebrew.
	if runtime.GOOS == "darwin" {
		if brewPath, err := exec.LookPath("brew"); err == nil {
			return installViaBrewAndCopy(ctx, brewPath, "imagemagick", v.dir, "convert",
				[]string{"convert", "magick"}, logCh)
		}
	}
	return fmt.Errorf("ImageMagick not found — install it first: %s", v.Instructions())
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

func (v *ImageMagickValidator) Version() string {
	p := v.ResolvedPath()
	if p == "" {
		return ""
	}
	out := runVersionCmd(p, "--version")
	line, _, _ := strings.Cut(out, "\n")
	// "Version: ImageMagick 7.1.1-..."
	if _, after, ok := strings.Cut(line, "ImageMagick "); ok {
		ver, _, _ := strings.Cut(after, " ")
		return ver
	}
	return ""
}

func (v *ImageMagickValidator) Status() ToolStatus {
	// Find system binary (convert or magick) and symlink as "convert"
	if systemPath := lookFirst("convert", "magick"); systemPath != "" {
		autoCopyBackground(v.dir, "convert", systemPath)
	}
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Version:   v.Version(),
		Required:  false,
	}
}

// ---------------------------------------------------------------------------
// BrewValidator
// ---------------------------------------------------------------------------

// BrewValidator checks for Homebrew on macOS (optional, not auto-installable).
type BrewValidator struct{}

func NewBrewValidator() *BrewValidator { return &BrewValidator{} }

func (v *BrewValidator) Name() string { return "brew" }

func (v *BrewValidator) IsInstalled() bool { return v.ResolvedPath() != "" }

func (v *BrewValidator) ResolvedPath() string {
	// Check the two canonical Homebrew locations before falling back to PATH.
	for _, candidate := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if statExists(candidate) {
			return candidate
		}
	}
	if p, err := exec.LookPath("brew"); err == nil {
		return p
	}
	return ""
}

func (v *BrewValidator) Version() string {
	p := v.ResolvedPath()
	if p == "" {
		return ""
	}
	out := runVersionCmd(p, "--version")
	// "Homebrew 4.x.y" — return the version number only.
	if _, after, ok := strings.Cut(out, "Homebrew "); ok {
		ver, _, _ := strings.Cut(after, "\n")
		return strings.TrimSpace(ver)
	}
	return ""
}

func (v *BrewValidator) Install(_ context.Context, _ chan<- string) error {
	return fmt.Errorf("Homebrew cannot be installed automatically — visit %s", v.Instructions())
}

func (v *BrewValidator) Instructions() string { return "https://brew.sh" }

func (v *BrewValidator) Status() ToolStatus {
	return ToolStatus{
		Name:      v.Name(),
		Installed: v.IsInstalled(),
		Path:      v.ResolvedPath(),
		Version:   v.Version(),
		Required:  false,
	}
}
