package configurator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// YtDlpValidator — required, auto-installable
// ---------------------------------------------------------------------------

type YtDlpValidator struct {
	baseValidator
	dir *AutoCutDir
}

func NewYtDlpValidator(dir *AutoCutDir, settings SettingRepo) *YtDlpValidator {
	path, src := resolveToolPath("yt-dlp", dir, settings)
	return &YtDlpValidator{
		baseValidator: baseValidator{
			name:       "yt-dlp",
			path:       path,
			required:   true,
			source:     src,
			binDir:     dir.BinDir,
			minVersion: "2023.01.01",
		},
		dir: dir,
	}
}

func (v *YtDlpValidator) Version() string {
	return v.versionWithArgs("--version")
}

func (v *YtDlpValidator) Install(ctx context.Context, logCh chan<- string) error {
	logCh <- "Detecting platform for yt-dlp download..."

	var downloadURL string
	switch runtime.GOOS {
	case "darwin":
		downloadURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_macos"
	case "linux":
		downloadURL = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	dest := v.dir.BinPath("yt-dlp")
	logCh <- fmt.Sprintf("Downloading yt-dlp from %s ...", downloadURL)

	if err := downloadFile(ctx, downloadURL, dest, logCh); err != nil {
		return fmt.Errorf("download yt-dlp: %w", err)
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return fmt.Errorf("chmod yt-dlp: %w", err)
	}
	v.path = dest
	slog.Info("yt-dlp installed", "path", dest)
	logCh <- "yt-dlp installed successfully."
	return nil
}

func (v *YtDlpValidator) Instructions() string {
	return "Visit https://github.com/yt-dlp/yt-dlp/releases to download yt-dlp, or use 'brew install yt-dlp'."
}

func (v *YtDlpValidator) LatestVersion(ctx context.Context) (string, error) {
	return resolveGitHubLatestTag(ctx, "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest")
}

func (v *YtDlpValidator) Status() ToolStatus {
	s := v.baseStatus()
	ver := v.Version()
	s.Version = ver
	if v.minVersion != "" {
		ok := checkMinVersion(ver, v.minVersion)
		s.VersionOK = &ok
	}
	return s
}

// ---------------------------------------------------------------------------
// FfmpegValidator — required, manual install
// ---------------------------------------------------------------------------

type FfmpegValidator struct {
	baseValidator
}

func NewFfmpegValidator(dir *AutoCutDir, settings SettingRepo) *FfmpegValidator {
	path, src := resolveToolPath("ffmpeg", dir, settings)
	return &FfmpegValidator{
		baseValidator: baseValidator{
			name:       "ffmpeg",
			path:       path,
			required:   true,
			source:     src,
			binDir:     dir.BinDir,
			minVersion: "5.0",
		},
	}
}

func (v *FfmpegValidator) Version() string {
	raw := v.versionWithArgs("-version")
	// ffmpeg -version first line: "ffmpeg version X ..."
	fields := strings.Fields(raw)
	if len(fields) >= 3 && fields[0] == "ffmpeg" && fields[1] == "version" {
		return fields[2]
	}
	return raw
}

func (v *FfmpegValidator) Install(_ context.Context, logCh chan<- string) error {
	logCh <- "ffmpeg requires manual installation."
	logCh <- v.Instructions()
	return fmt.Errorf("manual install required: %s", v.Instructions())
}

func (v *FfmpegValidator) Instructions() string {
	return "Install ffmpeg via Homebrew: 'brew install ffmpeg', or visit https://ffmpeg.org/download.html"
}

func (v *FfmpegValidator) Status() ToolStatus {
	s := v.baseStatus()
	ver := v.Version()
	s.Version = ver
	if v.minVersion != "" {
		ok := checkMinVersion(ver, v.minVersion)
		s.VersionOK = &ok
	}
	return s
}

// ---------------------------------------------------------------------------
// WhisperValidator — required, manual install
// ---------------------------------------------------------------------------

type WhisperValidator struct {
	baseValidator
	whisperDir string // ~/.autocut[-dev]/models/whisper
}

func NewWhisperValidator(dir *AutoCutDir, settings SettingRepo) *WhisperValidator {
	path, src := resolveToolPath("whisper-cli", dir, settings)
	return &WhisperValidator{
		baseValidator: baseValidator{
			name:       "whisper-cli",
			path:       path,
			required:   true,
			source:     src,
			binDir:     dir.BinDir,
			minVersion: "",
		},
		whisperDir: dir.WhisperDir,
	}
}

func (v *WhisperValidator) Version() string {
	// whisper-cli --help exits non-zero but prints version info
	raw := v.versionWithArgs("--help")
	if raw == "" || raw == "unknown" {
		return raw
	}
	return raw
}

func (v *WhisperValidator) Install(_ context.Context, logCh chan<- string) error {
	logCh <- "whisper-cli requires manual installation."
	logCh <- v.Instructions()
	return fmt.Errorf("manual install required: %s", v.Instructions())
}

func (v *WhisperValidator) Instructions() string {
	return "Build whisper.cpp from source: https://github.com/ggerganov/whisper.cpp"
}

// hasWhisperModel checks whether a ggml-*.bin model file exists in any of
// the standard candidate directories for whisper.cpp models.
func hasWhisperModel(binaryPath, binDir, whisperDir string) bool {
	homeDir, _ := os.UserHomeDir()
	candidates := []string{
		whisperDir,                                    // ~/.autocut[-dev]/models/whisper
		filepath.Dir(binaryPath),                      // same dir as binary
		filepath.Join(binDir, "models"),               // legacy subfolder
		filepath.Join(homeDir, ".cache", "whisper"),   // system cache
	}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(dir, "ggml-*.bin"))
		if err == nil && len(matches) > 0 {
			return true
		}
	}
	return false
}

func (v *WhisperValidator) Status() ToolStatus {
	s := v.baseStatus()
	if s.Installed && !hasWhisperModel(v.path, v.binDir, v.whisperDir) {
		slog.Warn("whisper-cli binary found but no model file detected", "path", v.path)
		s.Installed = false
		s.Source = "no_model"
	}
	ver := v.Version()
	s.Version = ver
	if v.minVersion != "" {
		ok := checkMinVersion(ver, v.minVersion)
		s.VersionOK = &ok
	}
	return s
}

// ---------------------------------------------------------------------------
// ClaudeValidator — optional, manual install
// ---------------------------------------------------------------------------

type ClaudeValidator struct {
	baseValidator
}

func NewClaudeValidator(dir *AutoCutDir, settings SettingRepo) *ClaudeValidator {
	path, src := resolveToolPath("claude", dir, settings)
	return &ClaudeValidator{
		baseValidator: baseValidator{
			name:       "claude",
			path:       path,
			required:   false,
			source:     src,
			binDir:     dir.BinDir,
			minVersion: "",
		},
	}
}

func (v *ClaudeValidator) Version() string {
	return v.versionWithArgs("--version")
}

func (v *ClaudeValidator) Install(_ context.Context, logCh chan<- string) error {
	logCh <- v.Instructions()
	return fmt.Errorf("manual install required: %s", v.Instructions())
}

func (v *ClaudeValidator) Instructions() string {
	return "Install Claude CLI: https://claude.ai/download"
}

func (v *ClaudeValidator) Status() ToolStatus {
	s := v.baseStatus()
	ver := v.Version()
	s.Version = ver
	if v.minVersion != "" {
		ok := checkMinVersion(ver, v.minVersion)
		s.VersionOK = &ok
	}
	return s
}

// ---------------------------------------------------------------------------
// GhValidator — optional, manual install
// ---------------------------------------------------------------------------

type GhValidator struct {
	baseValidator
}

func NewGhValidator(dir *AutoCutDir, settings SettingRepo) *GhValidator {
	path, src := resolveToolPath("gh", dir, settings)
	return &GhValidator{
		baseValidator: baseValidator{
			name:       "gh",
			path:       path,
			required:   false,
			source:     src,
			binDir:     dir.BinDir,
			minVersion: "",
		},
	}
}

func (v *GhValidator) Version() string {
	return v.versionWithArgs("--version")
}

func (v *GhValidator) Install(_ context.Context, logCh chan<- string) error {
	logCh <- v.Instructions()
	return fmt.Errorf("manual install required: %s", v.Instructions())
}

func (v *GhValidator) Instructions() string {
	return "Install GitHub CLI: 'brew install gh' or visit https://cli.github.com"
}

func (v *GhValidator) Status() ToolStatus {
	s := v.baseStatus()
	ver := v.Version()
	s.Version = ver
	if v.minVersion != "" {
		ok := checkMinVersion(ver, v.minVersion)
		s.VersionOK = &ok
	}
	return s
}

// ---------------------------------------------------------------------------
// OllamaValidator — optional, manual install; health check via HTTP
// ---------------------------------------------------------------------------

type OllamaValidator struct {
	baseValidator
	healthURL string
}

func NewOllamaValidator(dir *AutoCutDir, settings SettingRepo) *OllamaValidator {
	return newOllamaValidatorWithURL(dir, "http://localhost:11434", settings)
}

func newOllamaValidatorWithURL(dir *AutoCutDir, healthURL string, settings SettingRepo) *OllamaValidator {
	path, src := resolveToolPath("ollama", dir, settings)
	return &OllamaValidator{
		baseValidator: baseValidator{
			name:       "ollama",
			path:       path,
			required:   false,
			source:     src,
			binDir:     dir.BinDir,
			minVersion: "",
		},
		healthURL: healthURL,
	}
}

func (v *OllamaValidator) IsInstalled() bool {
	// Consider installed if binary is in ~/.autocut/bin (symlink OK) OR HTTP health passes.
	return v.baseValidator.IsInstalled() || v.httpHealthy()
}

func (v *OllamaValidator) httpHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.healthURL, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

func (v *OllamaValidator) Version() string {
	return v.versionWithArgs("--version")
}

func (v *OllamaValidator) Install(_ context.Context, logCh chan<- string) error {
	logCh <- v.Instructions()
	return fmt.Errorf("manual install required: %s", v.Instructions())
}

func (v *OllamaValidator) Instructions() string {
	return "Install Ollama from https://ollama.ai"
}

func (v *OllamaValidator) Status() ToolStatus {
	s := v.baseValidator.baseStatus()
	s.Installed = v.IsInstalled()
	ver := v.Version()
	s.Version = ver
	if v.minVersion != "" {
		ok := checkMinVersion(ver, v.minVersion)
		s.VersionOK = &ok
	}
	return s
}

// ---------------------------------------------------------------------------
// TwitchValidator — optional, auto-installable via GitHub releases
// ---------------------------------------------------------------------------

type TwitchValidator struct {
	baseValidator
	dir *AutoCutDir
}

func NewTwitchValidator(dir *AutoCutDir, settings SettingRepo) *TwitchValidator {
	path, src := resolveToolPath("TwitchDownloaderCLI", dir, settings)
	return &TwitchValidator{
		baseValidator: baseValidator{
			name:       "TwitchDownloaderCLI",
			path:       path,
			required:   true,
			source:     src,
			binDir:     dir.BinDir,
			minVersion: "1.50.0",
		},
		dir: dir,
	}
}

func (v *TwitchValidator) Version() string {
	return v.versionWithArgs("--help")
}

func (v *TwitchValidator) Install(ctx context.Context, logCh chan<- string) error {
	logCh <- "Resolving latest TwitchDownloaderCLI release from GitHub..."

	// Asset name pattern changed between releases (includes version number),
	// so we use the GitHub API to find the correct download URL dynamically.
	assetPattern := twitchAssetPattern()
	if assetPattern == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	downloadURL, err := resolveGitHubAssetURL(ctx,
		"https://api.github.com/repos/lay295/TwitchDownloader/releases/latest",
		assetPattern,
	)
	if err != nil {
		return fmt.Errorf("resolve TwitchDownloaderCLI release URL: %w", err)
	}
	logCh <- fmt.Sprintf("Found release asset: %s", downloadURL)

	dest := v.dir.BinPath("TwitchDownloaderCLI")
	zipPath := dest + ".zip"
	logCh <- fmt.Sprintf("Downloading TwitchDownloaderCLI...")

	if err := downloadFile(ctx, downloadURL, zipPath, logCh); err != nil {
		return fmt.Errorf("download TwitchDownloaderCLI: %w", err)
	}

	logCh <- "Extracting..."
	if err := unzipSingleBinary(zipPath, "TwitchDownloaderCLI", dest); err != nil {
		return fmt.Errorf("extract TwitchDownloaderCLI: %w", err)
	}
	if err := os.Remove(zipPath); err != nil {
		slog.Warn("failed to remove zip", "path", zipPath, "err", err)
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return fmt.Errorf("chmod TwitchDownloaderCLI: %w", err)
	}
	v.path = dest
	slog.Info("TwitchDownloaderCLI installed", "path", dest)
	logCh <- "TwitchDownloaderCLI installed successfully."
	return nil
}

// twitchAssetPattern returns the substring that identifies the correct release
// asset for the current platform/arch (matched against asset names from the API).
func twitchAssetPattern() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "MacOSArm64"
		}
		return "MacOS-x64"
	case "linux":
		return "Linux-x64"
	}
	return ""
}

func (v *TwitchValidator) Instructions() string {
	return "Download TwitchDownloaderCLI from https://github.com/lay295/TwitchDownloader/releases"
}

func (v *TwitchValidator) LatestVersion(ctx context.Context) (string, error) {
	return resolveGitHubLatestTag(ctx, "https://api.github.com/repos/lay295/TwitchDownloader/releases/latest")
}

func (v *TwitchValidator) Status() ToolStatus {
	s := v.baseStatus()
	ver := v.Version()
	s.Version = ver
	if v.minVersion != "" {
		ok := checkMinVersion(ver, v.minVersion)
		s.VersionOK = &ok
	}
	return s
}

// ---------------------------------------------------------------------------
// ConvertValidator — optional, manual install (ImageMagick)
// ---------------------------------------------------------------------------

type ConvertValidator struct {
	baseValidator
}

func NewConvertValidator(dir *AutoCutDir, settings SettingRepo) *ConvertValidator {
	path, src := resolveToolPath("convert", dir, settings)
	return &ConvertValidator{
		baseValidator: baseValidator{
			name:       "convert",
			path:       path,
			required:   false,
			source:     src,
			binDir:     dir.BinDir,
			minVersion: "",
		},
	}
}

func (v *ConvertValidator) Version() string {
	raw := v.versionWithArgs("--version")
	fields := strings.Fields(raw)
	if len(fields) >= 3 {
		return fields[2]
	}
	return raw
}

func (v *ConvertValidator) Install(_ context.Context, logCh chan<- string) error {
	logCh <- v.Instructions()
	return fmt.Errorf("manual install required: %s", v.Instructions())
}

func (v *ConvertValidator) Instructions() string {
	return "Install ImageMagick: 'brew install imagemagick' or visit https://imagemagick.org"
}

func (v *ConvertValidator) Status() ToolStatus {
	s := v.baseStatus()
	ver := v.Version()
	s.Version = ver
	if v.minVersion != "" {
		ok := checkMinVersion(ver, v.minVersion)
		s.VersionOK = &ok
	}
	return s
}

// ---------------------------------------------------------------------------
// downloadFile — shared HTTP download helper
// ---------------------------------------------------------------------------

func downloadFile(ctx context.Context, url, dest string, logCh chan<- string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dest), err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer f.Close()

	logCh <- fmt.Sprintf("Writing to %s ...", dest)
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return nil
}
