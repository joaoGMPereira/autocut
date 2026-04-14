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

func NewYtDlpValidator(dir *AutoCutDir) *YtDlpValidator {
	path := discoverBinary("yt-dlp", dir.BinDir)
	return &YtDlpValidator{
		baseValidator: baseValidator{name: "yt-dlp", path: path, required: true},
		dir:           dir,
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

func (v *YtDlpValidator) Status() ToolStatus {
	s := v.baseStatus()
	s.Version = v.Version()
	return s
}

// ---------------------------------------------------------------------------
// FfmpegValidator — required, manual install
// ---------------------------------------------------------------------------

type FfmpegValidator struct {
	baseValidator
}

func NewFfmpegValidator(dir *AutoCutDir) *FfmpegValidator {
	path := discoverBinary("ffmpeg", dir.BinDir)
	return &FfmpegValidator{
		baseValidator: baseValidator{name: "ffmpeg", path: path, required: true},
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
	s.Version = v.Version()
	return s
}

// ---------------------------------------------------------------------------
// WhisperValidator — required, manual install
// ---------------------------------------------------------------------------

type WhisperValidator struct {
	baseValidator
}

func NewWhisperValidator(dir *AutoCutDir) *WhisperValidator {
	path := discoverBinary("whisper-cli", dir.BinDir)
	return &WhisperValidator{
		baseValidator: baseValidator{name: "whisper-cli", path: path, required: true},
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

func (v *WhisperValidator) Status() ToolStatus {
	s := v.baseStatus()
	s.Version = v.Version()
	return s
}

// ---------------------------------------------------------------------------
// ClaudeValidator — optional, manual install
// ---------------------------------------------------------------------------

type ClaudeValidator struct {
	baseValidator
}

func NewClaudeValidator(dir *AutoCutDir) *ClaudeValidator {
	path := discoverBinary("claude", dir.BinDir)
	return &ClaudeValidator{
		baseValidator: baseValidator{name: "claude", path: path, required: false},
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
	s.Version = v.Version()
	return s
}

// ---------------------------------------------------------------------------
// GhValidator — optional, manual install
// ---------------------------------------------------------------------------

type GhValidator struct {
	baseValidator
}

func NewGhValidator(dir *AutoCutDir) *GhValidator {
	path := discoverBinary("gh", dir.BinDir)
	return &GhValidator{
		baseValidator: baseValidator{name: "gh", path: path, required: false},
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
	s.Version = v.Version()
	return s
}

// ---------------------------------------------------------------------------
// OllamaValidator — optional, manual install; health check via HTTP
// ---------------------------------------------------------------------------

type OllamaValidator struct {
	baseValidator
	healthURL string
}

func NewOllamaValidator(dir *AutoCutDir) *OllamaValidator {
	return newOllamaValidatorWithURL(dir, "http://localhost:11434")
}

func newOllamaValidatorWithURL(dir *AutoCutDir, healthURL string) *OllamaValidator {
	path := discoverBinary("ollama", dir.BinDir)
	return &OllamaValidator{
		baseValidator: baseValidator{name: "ollama", path: path, required: false},
		healthURL:     healthURL,
	}
}

func (v *OllamaValidator) IsInstalled() bool {
	// Consider installed if binary present OR HTTP health check passes.
	if v.path != "" {
		return true
	}
	return v.httpHealthy()
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
	s.Version = v.Version()
	return s
}

// ---------------------------------------------------------------------------
// TwitchValidator — optional, auto-installable via GitHub releases
// ---------------------------------------------------------------------------

type TwitchValidator struct {
	baseValidator
	dir *AutoCutDir
}

func NewTwitchValidator(dir *AutoCutDir) *TwitchValidator {
	path := discoverBinary("TwitchDownloaderCLI", dir.BinDir)
	return &TwitchValidator{
		baseValidator: baseValidator{name: "TwitchDownloaderCLI", path: path, required: false},
		dir:           dir,
	}
}

func (v *TwitchValidator) Version() string {
	return v.versionWithArgs("--help")
}

func (v *TwitchValidator) Install(ctx context.Context, logCh chan<- string) error {
	logCh <- "Detecting platform for TwitchDownloaderCLI download..."

	var downloadURL string
	switch runtime.GOOS {
	case "darwin":
		switch runtime.GOARCH {
		case "arm64":
			downloadURL = "https://github.com/lay295/TwitchDownloader/releases/latest/download/TwitchDownloaderCLI-Mac-Arm64.zip"
		default:
			downloadURL = "https://github.com/lay295/TwitchDownloader/releases/latest/download/TwitchDownloaderCLI-Mac-x64.zip"
		}
	case "linux":
		downloadURL = "https://github.com/lay295/TwitchDownloader/releases/latest/download/TwitchDownloaderCLI-Linux-x64.zip"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	dest := v.dir.BinPath("TwitchDownloaderCLI")
	zipPath := dest + ".zip"
	logCh <- fmt.Sprintf("Downloading TwitchDownloaderCLI from %s ...", downloadURL)

	if err := downloadFile(ctx, downloadURL, zipPath, logCh); err != nil {
		return fmt.Errorf("download TwitchDownloaderCLI: %w", err)
	}
	// Note: unzip is out of scope for minimal MVP — just mark the zip as downloaded
	// and point users to manual extraction. Full unzip implementation is a future enhancement.
	logCh <- fmt.Sprintf("Downloaded to %s. Extract TwitchDownloaderCLI binary to %s.", zipPath, v.dir.BinDir)
	slog.Info("TwitchDownloaderCLI zip downloaded", "path", zipPath)
	return nil
}

func (v *TwitchValidator) Instructions() string {
	return "Download TwitchDownloaderCLI from https://github.com/lay295/TwitchDownloader/releases"
}

func (v *TwitchValidator) Status() ToolStatus {
	s := v.baseStatus()
	s.Version = v.Version()
	return s
}

// ---------------------------------------------------------------------------
// ConvertValidator — optional, manual install (ImageMagick)
// ---------------------------------------------------------------------------

type ConvertValidator struct {
	baseValidator
}

func NewConvertValidator(dir *AutoCutDir) *ConvertValidator {
	path := discoverBinary("convert", dir.BinDir)
	return &ConvertValidator{
		baseValidator: baseValidator{name: "convert", path: path, required: false},
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
	s.Version = v.Version()
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
