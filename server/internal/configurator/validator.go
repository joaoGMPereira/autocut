package configurator

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// discoverBinary resolves a binary by name. It tries exec.LookPath first,
// then the AutoCut bin dir, then well-known Homebrew/system paths.
func discoverBinary(name string, extraPaths ...string) string {
	// 1. PATH resolution
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	// 2. Extra paths provided by caller (e.g. AutoCutDir.BinDir)
	for _, dir := range extraPaths {
		p := filepath.Join(dir, name)
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
		// Try the path directly (exec.LookPath needs PATH or abs path)
		if ok, _ := isExecutable(p); ok {
			return p
		}
	}

	// 3. Well-known Homebrew and system paths
	wellKnown := wellKnownDirs()
	for _, dir := range wellKnown {
		p := filepath.Join(dir, name)
		if ok, _ := isExecutable(p); ok {
			return p
		}
	}

	return ""
}

// wellKnownDirs returns platform-appropriate binary search directories.
func wellKnownDirs() []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"/opt/homebrew/bin",
			"/usr/local/bin",
			"/usr/bin",
			"/bin",
		}
	}
	return []string{
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
	}
}

// isExecutable returns true if the path exists and is executable.
func isExecutable(path string) (bool, error) {
	_, err := exec.LookPath(path)
	return err == nil, err
}

// baseValidator holds a resolved binary path and provides common ToolValidator helpers.
type baseValidator struct {
	name     string
	path     string
	required bool
}

// Name implements ToolValidator.
func (b *baseValidator) Name() string { return b.name }

// IsInstalled implements ToolValidator.
func (b *baseValidator) IsInstalled() bool { return b.path != "" }

// ResolvedPath implements ToolValidator.
func (b *baseValidator) ResolvedPath() string { return b.path }

// versionWithArgs runs the binary with the given args and returns trimmed output.
// Times out after 5s; returns "unknown" on timeout or error.
func (b *baseValidator) versionWithArgs(args ...string) string {
	if b.path == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, b.path, args...) //nolint:gosec
	out, err := cmd.CombinedOutput()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		slog.Warn("tool version check timed out", "tool", b.name)
		return "unknown"
	}
	if err != nil {
		// Some tools (e.g. whisper-cli --help) exit non-zero but still print useful info.
		// Treat non-empty output as success for version parsing.
		if len(out) == 0 {
			return ""
		}
	}

	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	return strings.TrimSpace(line)
}

// Status builds a ToolStatus from the base fields.
func (b *baseValidator) baseStatus() ToolStatus {
	path := b.path
	if path == "" {
		path = "unknown"
	}
	return ToolStatus{
		Name:      b.name,
		Installed: b.IsInstalled(),
		Required:  b.required,
		Path:      path,
	}
}
