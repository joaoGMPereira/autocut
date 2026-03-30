package thumbnail

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// FontInfo describes a single font found on the system.
// Kotlin ref: FontUtils.FontInfo
type FontInfo struct {
	// Name is the display name derived from the file name (without extension).
	Name string
	// Path is the absolute path to the font file.
	Path string
	// Family is the font family name (best-effort; may equal Name).
	Family string
	// Bold is true when the file name contains "bold" (case-insensitive).
	Bold bool
	// Italic is true when the file name contains "italic" (case-insensitive).
	Italic bool
}

// FontDetector discovers fonts available on the host system.
// Kotlin ref: FontDetectionDelegate
type FontDetector struct {
	log *slog.Logger
	exec Executor
}

// NewFontDetector returns a FontDetector backed by the real file system and
// real command execution.
func NewFontDetector() *FontDetector {
	return &FontDetector{
		log:  slog.With("component", "thumbnail.fonts"),
		exec: DefaultExecutor{},
	}
}

// newFontDetectorWithExec is used by tests to inject a mock executor.
func newFontDetectorWithExec(exec Executor) *FontDetector {
	return &FontDetector{
		log:  slog.With("component", "thumbnail.fonts"),
		exec: exec,
	}
}

// List returns all font files found on the current platform.
//
// macOS: reads /Library/Fonts/ and ~/Library/Fonts/
// Linux: tries fc-list; falls back to /usr/share/fonts/
// Windows: reads C:\Windows\Fonts\
//
// An empty slice is valid (e.g., on a CI runner without fonts).
// Kotlin ref: FontDetectionDelegate.findFromDirectPaths + fc-list usage
func (d *FontDetector) List() ([]FontInfo, error) {
	switch runtime.GOOS {
	case "darwin":
		return d.listDirs(macFontDirs())
	case "linux":
		fonts, err := d.listViaFcList()
		if err == nil && len(fonts) > 0 {
			return fonts, nil
		}
		// fc-list unavailable or returned nothing — fall back to common dirs
		d.log.Debug("fc-list unavailable, scanning fallback dirs", "err", err)
		return d.listDirs(linuxFallbackDirs())
	case "windows":
		return d.listDirs([]string{`C:\Windows\Fonts`})
	default:
		return nil, nil
	}
}

// Find performs a case-insensitive search for a font by name.
// Returns ErrFontNotFound when no match exists.
// Kotlin ref: FontDetectionDelegate.findAvailableFont
func (d *FontDetector) Find(name string) (*FontInfo, error) {
	fonts, err := d.List()
	if err != nil {
		return nil, fmt.Errorf("list fonts: %w", err)
	}
	lower := strings.ToLower(name)
	for i := range fonts {
		if strings.ToLower(fonts[i].Name) == lower {
			return &fonts[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrFontNotFound, name)
}

// SystemDefault returns a font name that is virtually guaranteed to exist
// on the current platform.
// Kotlin ref: FontDetectionDelegate.preferredFontNames
func (d *FontDetector) SystemDefault() string {
	switch runtime.GOOS {
	case "linux":
		return "DejaVuSans"
	default: // darwin + windows
		return "Arial"
	}
}

// ErrFontNotFound is returned by Find when no matching font exists.
var ErrFontNotFound = fmt.Errorf("font not found")

// --- internal helpers ---

func macFontDirs() []string {
	dirs := []string{"/Library/Fonts"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
	}
	return dirs
}

func linuxFallbackDirs() []string {
	return []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
	}
}

// listDirs scans a set of directories (non-recursive for top-level, recursive
// via filepath.WalkDir for nested structures) and collects font files.
func (d *FontDetector) listDirs(dirs []string) ([]FontInfo, error) {
	var fonts []FontInfo
	seen := map[string]struct{}{}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			// Directory may simply not exist on this machine — not an error.
			d.log.Debug("font dir not readable", "dir", dir, "err", err)
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				// Recurse one level for organised font directories.
				sub := filepath.Join(dir, e.Name())
				subFonts, _ := d.listDirs([]string{sub})
				for _, f := range subFonts {
					if _, ok := seen[f.Path]; !ok {
						seen[f.Path] = struct{}{}
						fonts = append(fonts, f)
					}
				}
				continue
			}
			path := filepath.Join(dir, e.Name())
			if _, ok := seen[path]; ok {
				continue
			}
			if info := fontInfoFromPath(path); info != nil {
				seen[path] = struct{}{}
				fonts = append(fonts, *info)
			}
		}
	}
	return fonts, nil
}

// listViaFcList uses the fc-list command available on most Linux desktops.
// Output format requested: "<fullname>\t<file>"
// Kotlin ref: no direct equivalent — Linux-specific addition
func (d *FontDetector) listViaFcList(args ...string) ([]FontInfo, error) {
	out, err := d.exec.Run("fc-list", "--format=%{fullname}\t%{file}\n")
	if err != nil {
		return nil, fmt.Errorf("fc-list: %w", err)
	}

	var fonts []FontInfo
	seen := map[string]struct{}{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[1])
		if name == "" || path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		lname := strings.ToLower(name)
		fonts = append(fonts, FontInfo{
			Name:   name,
			Path:   path,
			Family: name,
			Bold:   strings.Contains(lname, "bold"),
			Italic: strings.Contains(lname, "italic") || strings.Contains(lname, "oblique"),
		})
	}
	return fonts, nil
}

// fontInfoFromPath builds a FontInfo for a single font file path, or returns
// nil if the file does not look like a font.
func fontInfoFromPath(path string) *FontInfo {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ttf", ".otf", ".ttc", ".otc":
		// recognised font formats
	default:
		return nil
	}
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	lower := strings.ToLower(name)
	return &FontInfo{
		Name:   name,
		Path:   path,
		Family: name,
		Bold:   strings.Contains(lower, "bold"),
		Italic: strings.Contains(lower, "italic") || strings.Contains(lower, "oblique"),
	}
}
