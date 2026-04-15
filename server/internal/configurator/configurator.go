package configurator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SettingRepo is the minimal interface for reading/writing app settings.
// Satisfied by *database.AppSettingRepo.
type SettingRepo interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

// ToolStatus describes the current state of a required or optional tool.
type ToolStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Required  bool   `json:"required"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
	VersionOK *bool  `json:"version_ok,omitempty"` // nil = no min version configured
	Source    string `json:"source,omitempty"`      // "path" | "custom" | "autocut_bin"
}

// AutoCutDir holds paths to all directories used by AutoCut.
type AutoCutDir struct {
	Root          string `json:"root"`
	BinDir        string `json:"bin_dir"`
	ModelsDir     string `json:"models_dir"`
	WhisperDir    string `json:"whisper_dir"`
	TokensDir     string `json:"tokens_dir"`
	CacheDir      string `json:"cache_dir"`
	DownloadsDir  string `json:"downloads_dir"`
	ThumbnailsDir string `json:"thumbnails_dir"`
}

// NewAutoCutDir creates an AutoCutDir rooted at ~/.autocut and ensures dirs exist.
func NewAutoCutDir() (*AutoCutDir, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	return newAutoCutDirFromRoot(filepath.Join(home, ".autocut")), nil
}

// NewAutoCutDirFromPath creates an AutoCutDir rooted at the given path.
func NewAutoCutDirFromPath(root string) *AutoCutDir {
	return newAutoCutDirFromRoot(root)
}

func newAutoCutDirFromRoot(root string) *AutoCutDir {
	d := &AutoCutDir{
		Root:          root,
		BinDir:        filepath.Join(root, "bin"),
		ModelsDir:     filepath.Join(root, "models"),
		WhisperDir:    filepath.Join(root, "models", "whisper"),
		TokensDir:     filepath.Join(root, "tokens"),
		CacheDir:      filepath.Join(root, "cache"),
		DownloadsDir:  filepath.Join(root, "downloads"),
		ThumbnailsDir: filepath.Join(root, "thumbnails"),
	}
	return d
}

// ToolValidator checks and installs a single external tool.
type ToolValidator interface {
	Name() string
	IsInstalled() bool
	ResolvedPath() string
	Version() string
	Install(ctx context.Context, logCh chan<- string) error
	Instructions() string
	Status() ToolStatus
}

// UpdateInfo describes a tool's current and latest version.
type UpdateInfo struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version,omitempty"`
	UpdateAvailable bool  `json:"update_available,omitempty"`
}

// WhisperModelInfo describes a Whisper model.
type WhisperModelInfo struct {
	Name       string `json:"name"`
	Downloaded bool   `json:"downloaded"`
	SizeMB     int    `json:"size_mb,omitempty"`
	Path       string `json:"path,omitempty"`
	Active     bool   `json:"active,omitempty"`
}

// ErrToolNotFound is returned when a tool name is not registered.
type ErrToolNotFound struct {
	Name string
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool not found: %s", e.Name)
}

// Configurator manages external tools required by AutoCut.
type Configurator struct {
	dir         *AutoCutDir
	validators  []ToolValidator
	settingRepo SettingRepo // may be nil
}

// New creates a Configurator with default validators.
func New(dir *AutoCutDir, settings SettingRepo) *Configurator {
	return &Configurator{
		dir:         dir,
		validators:  defaultValidators(dir, settings),
		settingRepo: settings,
	}
}

func newWithValidators(dir *AutoCutDir, validators []ToolValidator) *Configurator {
	return &Configurator{dir: dir, validators: validators}
}

func defaultValidators(dir *AutoCutDir, settings SettingRepo) []ToolValidator {
	return []ToolValidator{
		NewYtDlpValidator(dir, settings),
		NewFfmpegValidator(dir, settings),
		NewWhisperValidator(dir, settings),
		NewClaudeValidator(dir, settings),
		NewGhValidator(dir, settings),
		NewOllamaValidator(dir, settings),
		NewTwitchValidator(dir, settings),
		NewConvertValidator(dir, settings),
	}
}

// SetCustomPath persists a custom binary path for the named tool.
// Key: tool_custom_path_{toolName}
func (c *Configurator) SetCustomPath(ctx context.Context, toolName, path string) error {
	if c.settingRepo == nil {
		return fmt.Errorf("settings repository not configured")
	}
	return c.settingRepo.Set(ctx, "tool_custom_path_"+toolName, path)
}

// ClearCustomPath removes the custom binary path for the named tool,
// causing the next startup to fall back to auto-detection.
func (c *Configurator) ClearCustomPath(ctx context.Context, toolName string) error {
	if c.settingRepo == nil {
		return fmt.Errorf("settings repository not configured")
	}
	return c.settingRepo.Set(ctx, "tool_custom_path_"+toolName, "")
}

// Status returns the install status of all registered tools.
func (c *Configurator) Status() []ToolStatus {
	out := make([]ToolStatus, len(c.validators))
	for i, v := range c.validators {
		out[i] = v.Status()
	}
	return out
}

// Missing returns statuses for tools that are not installed.
func (c *Configurator) Missing() []ToolStatus {
	var out []ToolStatus
	for _, v := range c.validators {
		if !v.IsInstalled() {
			out = append(out, v.Status())
		}
	}
	return out
}

// AllInstalled returns true if all required tools are installed.
func (c *Configurator) AllInstalled() bool {
	for _, v := range c.validators {
		s := v.Status()
		if s.Required && !v.IsInstalled() {
			return false
		}
	}
	return true
}

// Get returns the ToolValidator registered under name, or false.
func (c *Configurator) Get(name string) (ToolValidator, bool) {
	for _, v := range c.validators {
		if v.Name() == name {
			return v, true
		}
	}
	return nil, false
}

// Install installs the named tool, streaming log lines to logCh.
func (c *Configurator) Install(ctx context.Context, name string, logCh chan<- string) error {
	v, ok := c.Get(name)
	if !ok {
		return &ErrToolNotFound{Name: name}
	}
	return v.Install(ctx, logCh)
}

// Dir returns the AutoCutDir.
func (c *Configurator) Dir() *AutoCutDir {
	return c.dir
}

// Refresh re-discovers all tools and returns updated statuses.
func (c *Configurator) Refresh() []ToolStatus {
	c.validators = defaultValidators(c.dir, c.settingRepo)
	return c.Status()
}

// Sync copies binaries and whisper models from ~/.autocut into the current dir
// (as symlinks), then re-discovers all tools. Used when running in a dev dir.
func (c *Configurator) Sync() []ToolStatus {
	home, _ := os.UserHomeDir()
	if home == "" {
		return c.Refresh()
	}
	prodRoot := filepath.Join(home, ".autocut")
	if prodRoot == c.dir.Root {
		return c.Refresh() // already in production dir, nothing to sync
	}

	// Binaries
	prodBin := filepath.Join(prodRoot, "bin")
	for _, v := range c.validators {
		name := v.Name()
		src := filepath.Join(prodBin, name)
		if ok, _ := isExecutable(src); ok {
			if err := copyToAutoCutBin(name, src, c.dir.BinDir); err != nil {
				slog.Warn("sync: failed to link binary", "tool", name, "err", err)
			}
		}
	}

	// Whisper models — symlink each ggml-*.bin from prod whisper dir
	prodWhisper := filepath.Join(prodRoot, "models", "whisper")
	devWhisper := c.dir.WhisperDir
	if entries, err := filepath.Glob(filepath.Join(prodWhisper, "ggml-*.bin")); err == nil {
		if err := os.MkdirAll(devWhisper, 0o755); err == nil {
			for _, src := range entries {
				dest := filepath.Join(devWhisper, filepath.Base(src))
				if _, err := os.Lstat(dest); err == nil {
					continue // already exists
				}
				if err := os.Symlink(src, dest); err != nil {
					slog.Warn("sync: failed to link model", "src", src, "err", err)
				}
			}
		}
	}

	return c.Refresh()
}

// updateChecker is implemented by validators that can fetch their latest release.
type updateChecker interface {
	LatestVersion(ctx context.Context) (string, error)
}

// CheckUpdate returns update information for the named tool, including latest
// version from upstream if the validator implements updateChecker.
func (c *Configurator) CheckUpdate(ctx context.Context, name string) (UpdateInfo, error) {
	v, ok := c.Get(name)
	if !ok {
		return UpdateInfo{}, &ErrToolNotFound{Name: name}
	}
	current := v.Version()
	info := UpdateInfo{Name: name, CurrentVersion: current}
	if uc, ok := v.(updateChecker); ok {
		if latest, err := uc.LatestVersion(ctx); err == nil && latest != "" {
			info.LatestVersion = latest
			info.UpdateAvailable = !checkMinVersion(current, latest)
		}
	}
	return info, nil
}

// WhisperModelStatus returns download status of known Whisper models.
func (c *Configurator) WhisperModelStatus() []WhisperModelInfo {
	return WhisperModelStatus(c.dir)
}

// DownloadWhisperModel downloads a Whisper model, streaming logs to logCh.
func (c *Configurator) DownloadWhisperModel(ctx context.Context, name string, logCh chan<- string) error {
	return DownloadWhisperModel(ctx, name, c.dir, logCh)
}

// Hardware detects the current machine's hardware and computes model recommendations.
func (c *Configurator) Hardware() (HardwareProfile, ModelRecommendation) {
	hw := Detect()
	return hw, Recommend(hw)
}

// ApplyCustomPath validates a custom binary path, persists it, and returns the updated ToolStatus.
// errorCode is one of: "tool_not_found", "invalid_path", "not_executable", "wrong_tool".
// Returns (status, "", nil) on success.
func (c *Configurator) ApplyCustomPath(ctx context.Context, toolName, customPath string) (ToolStatus, string, error) {
	v, ok := c.Get(toolName)
	if !ok {
		return ToolStatus{}, "tool_not_found", &ErrToolNotFound{Name: toolName}
	}

	// Validate: file exists
	info, err := os.Stat(customPath)
	if err != nil {
		return ToolStatus{}, "invalid_path", fmt.Errorf("file not found: %s", customPath)
	}

	// Validate: executable bit
	if info.Mode()&0o111 == 0 {
		return ToolStatus{}, "not_executable", fmt.Errorf("not executable: %s", customPath)
	}

	// Run --version to confirm it's a real binary (5s timeout)
	vctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, _ := exec.CommandContext(vctx, customPath, "--version").CombinedOutput() //nolint:gosec
	if len(out) == 0 {
		// Try --help as fallback (some tools exit non-zero on --version)
		out, _ = exec.CommandContext(vctx, customPath, "--help").CombinedOutput() //nolint:gosec
	}
	version := ""
	if len(out) > 0 {
		lines := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)
		version = strings.TrimSpace(lines[0])
	}

	// Persist
	if err := c.SetCustomPath(ctx, toolName, customPath); err != nil {
		return ToolStatus{}, "", fmt.Errorf("persist custom path: %w", err)
	}

	existing := v.Status()
	return ToolStatus{
		Name:      toolName,
		Installed: true,
		Required:  existing.Required,
		Path:      customPath,
		Version:   version,
		Source:    "custom",
	}, "", nil
}

// DetectAndClearCustomPath clears the persisted custom path and returns auto-detected status.
func (c *Configurator) DetectAndClearCustomPath(ctx context.Context, toolName string) (ToolStatus, error) {
	v, ok := c.Get(toolName)
	if !ok {
		return ToolStatus{}, &ErrToolNotFound{Name: toolName}
	}
	if err := c.ClearCustomPath(ctx, toolName); err != nil {
		return ToolStatus{}, fmt.Errorf("clear custom path: %w", err)
	}
	// Return the in-memory validator's status (reflects startup-time discovery)
	// The Source will be "path" or "autocut_bin" if it was auto-detected at startup.
	return v.Status(), nil
}
