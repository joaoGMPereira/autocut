package configurator

import "context"

// Configurator is the facade over all ToolValidator instances.
type Configurator struct {
	dir        *AutoCutDir
	validators []ToolValidator
}

// New creates a Configurator with the default six validators in canonical order:
// YtDlp → TwitchCLI → FFmpeg → Whisper → Ollama → ImageMagick.
func New(dir *AutoCutDir) *Configurator {
	return &Configurator{
		dir: dir,
		validators: []ToolValidator{
			NewYtDlpValidator(dir),
			NewTwitchCLIValidator(dir),
			NewFFmpegValidator(dir),
			NewWhisperValidator(dir),
			NewOllamaValidator(dir),
			NewImageMagickValidator(dir),
		},
	}
}

// newWithValidators creates a Configurator with a custom validator list.
// Used in tests to inject mocks.
func newWithValidators(dir *AutoCutDir, validators []ToolValidator) *Configurator {
	return &Configurator{dir: dir, validators: validators}
}

// Status returns the ToolStatus of every registered validator.
func (c *Configurator) Status() []ToolStatus {
	out := make([]ToolStatus, len(c.validators))
	for i, v := range c.validators {
		out[i] = v.Status()
	}
	return out
}

// Required returns the status of every required tool.
func (c *Configurator) Required() []ToolStatus {
	var out []ToolStatus
	for _, v := range c.validators {
		if s := v.Status(); s.Required {
			out = append(out, s)
		}
	}
	return out
}

// Missing returns the status of every tool that is not installed.
func (c *Configurator) Missing() []ToolStatus {
	var out []ToolStatus
	for _, v := range c.validators {
		if s := v.Status(); !s.Installed {
			out = append(out, s)
		}
	}
	return out
}

// Get returns the validator with the given name, or (nil, false) if not found.
func (c *Configurator) Get(name string) (ToolValidator, bool) {
	for _, v := range c.validators {
		if v.Name() == name {
			return v, true
		}
	}
	return nil, false
}

// Install delegates to the named validator's Install method.
func (c *Configurator) Install(ctx context.Context, name string, logCh chan<- string) error {
	v, ok := c.Get(name)
	if !ok {
		return &ErrToolNotFound{Name: name}
	}
	return v.Install(ctx, logCh)
}

// ResolvedPaths returns a map of tool name → resolved path for every validator.
// Tools that are not installed map to an empty string.
func (c *Configurator) ResolvedPaths() map[string]string {
	out := make(map[string]string, len(c.validators))
	for _, v := range c.validators {
		out[v.Name()] = v.ResolvedPath()
	}
	return out
}

// AllInstalled returns true when every required tool is installed.
func (c *Configurator) AllInstalled() bool {
	for _, v := range c.validators {
		if s := v.Status(); s.Required && !s.Installed {
			return false
		}
	}
	return true
}

// Dir returns the AutoCutDir used by this Configurator.
func (c *Configurator) Dir() *AutoCutDir {
	return c.dir
}

// ErrToolNotFound is returned when Install is called with an unknown tool name.
type ErrToolNotFound struct {
	Name string
}

func (e *ErrToolNotFound) Error() string {
	return "configurator: tool not found: " + e.Name
}
