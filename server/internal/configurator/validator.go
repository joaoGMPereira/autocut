package configurator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// loginShellPATHDirs returns the PATH directories from the user's login shell.
// On macOS, GUI apps launched from Finder/Dock get a restricted launchd PATH that
// excludes Homebrew (/opt/homebrew/bin). The login shell sources ~/.zprofile which
// runs `eval "$(brew shellenv)"`, giving us the full user-configured PATH.
// This is the same technique used by VS Code and IntelliJ for macOS GUI launches.
func loginShellPATHDirs() []string {
	if runtime.GOOS != "darwin" {
		return nil
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// -l = login shell (sources ~/.zprofile, no oh-my-zsh/.zshrc overhead)
	out, err := exec.CommandContext(ctx, shell, "-l", "-c", `printf "%s" "$PATH"`).Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	var dirs []string
	for _, d := range strings.Split(string(out), string(os.PathListSeparator)) {
		if d != "" {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// AugmentPATH prepends well-known binary directories — plus any extraDirs — to
// the process PATH so exec.LookPath and exec.Command find tools installed by
// Homebrew, in /usr/local/bin, or in the app's own bin dir (~/.autocut/bin)
// even when launched with a restricted PATH (e.g. from Electron on macOS).
// Safe to call multiple times — deduplicates the prepended dirs.
func AugmentPATH(extraDirs ...string) {
	var dirs []string
	dirs = append(dirs, extraDirs...)
	dirs = append(dirs, loginShellPATHDirs()...)
	dirs = append(dirs, wellKnownDirs()...)
	existing := os.Getenv("PATH")
	existingSet := make(map[string]bool)
	for _, p := range strings.Split(existing, string(os.PathListSeparator)) {
		existingSet[p] = true
	}
	var toAdd []string
	seen := make(map[string]bool)
	for _, d := range dirs {
		if d != "" && !existingSet[d] && !seen[d] {
			toAdd = append(toAdd, d)
			seen[d] = true
		}
	}
	if len(toAdd) == 0 {
		return
	}
	os.Setenv("PATH", strings.Join(append(toAdd, existing), string(os.PathListSeparator)))
}

// FindBinary returns the absolute path of name by checking binDir then
// well-known system directories using direct file access — no PATH lookup.
// Works even when the process has a restricted PATH (e.g. macOS GUI app).
func FindBinary(name, binDir string) string {
	return discoverBinary(name, binDir)
}

// brewBin returns the absolute path to the brew binary without relying on PATH.
func brewBin() string {
	for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
		if fi, err := os.Stat(p); err == nil && fi.Mode()&0111 != 0 {
			return p
		}
	}
	return ""
}

// discoverBinary resolves a binary by name using the following priority:
//  1. Extra paths (BinDir) — tools explicitly placed in ~/.autocut/bin win
//  2. Default ~/.autocut/bin fallback — finds production binaries when running
//     from a dev dir (e.g. ~/.autocut-dev/bin)
//  3. System PATH
//  4. Well-known Homebrew/system paths
func discoverBinary(name string, extraPaths ...string) string {
	// 1. Extra paths first (BinDir takes precedence)
	for _, dir := range extraPaths {
		p := filepath.Join(dir, name)
		if ok, _ := isExecutable(p); ok {
			return p
		}
	}

	// 2. System PATH
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	// 4. Well-known Homebrew and system paths
	for _, dir := range wellKnownDirs() {
		p := filepath.Join(dir, name)
		if ok, _ := isExecutable(p); ok {
			return p
		}
	}

	return ""
}

// wellKnownDirs returns platform-appropriate binary search directories.
// User-local dirs (~/bin, ~/.local/bin) are prepended so user-installed binaries
// take precedence over system-wide ones (FR-002).
func wellKnownDirs() []string {
	home, _ := os.UserHomeDir()
	var homeDirs []string
	if home != "" {
		homeDirs = []string{
			filepath.Join(home, "bin"),
			filepath.Join(home, ".local", "bin"),
		}
	}
	if runtime.GOOS == "darwin" {
		return append(homeDirs, []string{
			"/opt/homebrew/bin",
			"/usr/local/bin",
			"/usr/bin",
			"/bin",
		}...)
	}
	return append(homeDirs, []string{
		"/usr/local/bin",
		"/usr/bin",
		"/bin",
	}...)
}

// isExecutable returns true if the path exists and is executable.
func isExecutable(path string) (bool, error) {
	_, err := exec.LookPath(path)
	return err == nil, err
}

// copyToAutoCutBin creates a symlink in the AutoCut bin dir pointing to resolvedPath.
// It skips if a valid symlink to the same target already exists.
// If a stale symlink exists (different target), it removes and recreates it.
func copyToAutoCutBin(name, resolvedPath, binDir string) error {
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create autocut bin dir: %w", err)
	}
	dest := filepath.Join(binDir, name)

	if existing, err := os.Readlink(dest); err == nil {
		// dest is a symlink
		if existing == resolvedPath {
			return nil // already correct
		}
		// Stale symlink — remove and recreate
		if err := os.Remove(dest); err != nil {
			return fmt.Errorf("remove stale symlink %s: %w", dest, err)
		}
	} else if _, statErr := os.Stat(dest); statErr == nil {
		// dest is a regular file (real binary already installed here — don't overwrite)
		return nil
	}

	if err := os.Symlink(resolvedPath, dest); err != nil {
		return fmt.Errorf("create symlink %s -> %s: %w", dest, resolvedPath, err)
	}
	// Remove quarantine from the symlink — macOS may block execution of
	// quarantined binaries when spawned from a hardened-runtime process.
	exec.Command("xattr", "-d", "com.apple.quarantine", dest).Run() //nolint:errcheck,gosec
	return nil
}

// detectSource resolves source classification for a discovered binary.
// When the binary is found via system PATH, it creates a symlink in the AutoCut
// bin dir and returns the symlink path so all tools are rooted at ~/.autocut/bin.
// Returns (resolvedPath, source).
func detectSource(name, resolvedPath, binDir string) (string, string) {
	if resolvedPath == "" {
		return "", ""
	}
	// Already in autocut bin dir — clear quarantine in case it was set
	autoCutBinPath := filepath.Join(binDir, name)
	if resolvedPath == autoCutBinPath {
		exec.Command("xattr", "-d", "com.apple.quarantine", resolvedPath).Run() //nolint:errcheck,gosec
		return resolvedPath, "autocut_bin"
	}
	// Found via system — create symlink and expose the autocut bin path
	if err := copyToAutoCutBin(name, resolvedPath, binDir); err != nil {
		slog.Warn("failed to symlink tool to autocut bin", "tool", name, "err", err)
		// Fallback: use system path but mark as "path" (will not satisfy gate)
		return resolvedPath, "path"
	}
	return autoCutBinPath, "autocut_bin"
}

// resolveToolPath resolves a tool's binary path, checking for a custom path first.
// Custom paths take precedence over auto-discovery.
func resolveToolPath(name string, dir *AutoCutDir, settings SettingRepo) (path, source string) {
	if settings != nil {
		customPath, _ := settings.Get(context.Background(), "tool_custom_path_"+name)
		if customPath != "" {
			if ok, _ := isExecutable(customPath); ok {
				return customPath, "custom"
			}
			// Invalid custom path — log and fall through to auto-discovery
			slog.Warn("custom tool path invalid, falling back to auto-discovery",
				"tool", name, "path", customPath)
		}
	}
	discovered := discoverBinary(name, dir.BinDir)
	return detectSource(name, discovered, dir.BinDir)
}

// baseValidator holds a resolved binary path and provides common ToolValidator helpers.
type baseValidator struct {
	name       string
	path       string
	required   bool
	source     string
	binDir     string
	minVersion string // empty = no minimum version enforced
}

// Name implements ToolValidator.
func (b *baseValidator) Name() string { return b.name }

// IsInstalled implements ToolValidator.
// Returns true only when the tool is confirmed at ~/.autocut/bin (source "autocut_bin")
// or at a user-specified custom path (source "custom").
// A tool found only in system PATH (source "path") does not satisfy the gate.
func (b *baseValidator) IsInstalled() bool {
	return b.source == "autocut_bin" || b.source == "custom"
}

// ResolvedPath implements ToolValidator.
func (b *baseValidator) ResolvedPath() string { return b.path }

// versionWithArgs runs the binary with the given args and returns trimmed output.
// Times out after 8s; returns "unknown" on timeout or error.
func (b *baseValidator) versionWithArgs(args ...string) string {
	if b.path == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
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
		Source:    b.source,
	}
}

// checkMinVersion returns true when detectedVersion satisfies the minimum.
// Conservative rules:
//   - minimum is empty → always true (no minimum enforced)
//   - detected is empty/unknown → true (can't compare, don't block)
//   - non-numeric detected (e.g. "N-12345") → true (conservative)
//   - otherwise: numeric component-by-component comparison
func checkMinVersion(detected, minimum string) bool {
	if minimum == "" {
		return true
	}
	if detected == "" || detected == "unknown" {
		return true
	}

	detNorm := extractVersion(detected)
	minNorm := extractVersion(minimum)

	if detNorm == "" {
		// Non-numeric detected version — conservative: assume OK
		return true
	}

	detParts := strings.Split(detNorm, ".")
	minParts := strings.Split(minNorm, ".")

	// Pad to equal length
	for len(detParts) < len(minParts) {
		detParts = append(detParts, "0")
	}
	for len(minParts) < len(detParts) {
		minParts = append(minParts, "0")
	}

	for i := range minParts {
		d, errD := strconv.Atoi(detParts[i])
		m, errM := strconv.Atoi(minParts[i])
		if errD != nil || errM != nil {
			// Non-numeric component — conservative
			return true
		}
		if d > m {
			return true
		}
		if d < m {
			return false
		}
		// equal — check next component
	}
	return true // equal → satisfies minimum
}

// normalizeVersion trims leading 'v', then returns only the leading numeric+dot prefix.
// e.g. "v5.1.2-alpha" → "5.1.2", "2023.01.04" → "2023.01.04", "N-12345" → ""
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	end := 0
	for end < len(v) && (v[end] == '.' || (v[end] >= '0' && v[end] <= '9')) {
		end++
	}
	return v[:end]
}

// extractVersion finds the first numeric version segment in s.
// e.g. "TwitchDownloaderCLI 1.56.4+hash" → "1.56.4", "2026.03.17" → "2026.03.17"
func extractVersion(s string) string {
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return normalizeVersion(s[i:])
		}
	}
	return ""
}
