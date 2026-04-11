package configurator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ToolStatus describes the current state of a required or optional tool.
type ToolStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Required  bool   `json:"required"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
}

// AutoCutDir holds paths to all directories used by AutoCut.
type AutoCutDir struct {
	Root          string `json:"root"`
	BinDir        string `json:"bin_dir"`
	ModelsDir     string `json:"models_dir"`
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

func newAutoCutDirFromRoot(root string) *AutoCutDir {
	d := &AutoCutDir{
		Root:          root,
		BinDir:        filepath.Join(root, "bin"),
		ModelsDir:     filepath.Join(root, "models"),
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
	Name        string `json:"name"`
	Downloaded  bool   `json:"downloaded"`
	SizeMB      int    `json:"size_mb,omitempty"`
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
	dir        *AutoCutDir
	validators []ToolValidator
}

// New creates a Configurator with default validators.
func New(dir *AutoCutDir) *Configurator {
	return &Configurator{
		dir:        dir,
		validators: defaultValidators(dir),
	}
}

func newWithValidators(dir *AutoCutDir, validators []ToolValidator) *Configurator {
	return &Configurator{dir: dir, validators: validators}
}

func defaultValidators(_ *AutoCutDir) []ToolValidator {
	return []ToolValidator{}
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

// CheckUpdate returns update information for the named tool.
func (c *Configurator) CheckUpdate(_ context.Context, name string) (UpdateInfo, error) {
	v, ok := c.Get(name)
	if !ok {
		return UpdateInfo{}, &ErrToolNotFound{Name: name}
	}
	return UpdateInfo{Name: name, CurrentVersion: v.Version()}, nil
}

// WhisperModelStatus returns download status of known Whisper models.
func (c *Configurator) WhisperModelStatus() []WhisperModelInfo {
	return nil
}

// DownloadWhisperModel downloads a Whisper model, streaming logs to logCh.
func (c *Configurator) DownloadWhisperModel(_ context.Context, _ string, logCh chan<- string) error {
	logCh <- "not implemented"
	return nil
}
