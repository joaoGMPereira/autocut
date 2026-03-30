package configurator

import "context"

// ToolStatus is the serialisable state of a single tool check.
type ToolStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
	Required  bool   `json:"required"`
}

// ToolValidator is the strategy interface for every external-tool check.
type ToolValidator interface {
	// Name returns the canonical tool name (e.g. "yt-dlp").
	Name() string

	// IsInstalled returns true when the tool is usable on this system.
	IsInstalled() bool

	// ResolvedPath returns the absolute path that will be used to invoke the tool.
	// Returns an empty string when the tool is not installed.
	ResolvedPath() string

	// Install attempts an automated installation, streaming progress to logCh.
	// Returns a non-nil error when installation fails or is not supported.
	Install(ctx context.Context, logCh chan<- string) error

	// Instructions returns a human-readable string describing manual installation steps.
	Instructions() string

	// Status returns a snapshot of the current tool state.
	Status() ToolStatus
}
