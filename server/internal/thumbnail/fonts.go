package thumbnail

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// fontDirs returns the OS-specific directories to scan for fonts.
func fontDirs() []string {
	if runtime.GOOS == "darwin" {
		return []string{
			"/Library/Fonts",
			"/System/Library/Fonts",
			"/System/Library/Fonts/Supplemental",
		}
	}
	// Linux
	return []string{"/usr/share/fonts"}
}

// fallbackChain is the preferred font order when the requested font is not found.
var fallbackChain = []string{"Impact", "Arial Bold", "Arial", "Helvetica"}

// DetectFonts walks system font directories and returns all .ttf/.ttc files
// grouped by family name (filename without extension).
func DetectFonts() []FontInfo {
	log := slog.With("component", "thumbnail.fonts")
	seen := make(map[string]bool)
	var fonts []FontInfo

	for _, dir := range fontDirs() {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip unreadable dirs
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".ttf" && ext != ".ttc" {
				return nil
			}
			family := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			if seen[family] {
				return nil
			}
			seen[family] = true
			fonts = append(fonts, FontInfo{Family: family, Path: path})
			return nil
		})
	}

	log.Info("font detection complete", "count", len(fonts))
	return fonts
}

// ResolveFont finds the font file path for the given family name.
// Falls back through the fallback chain if the requested font is not found.
// Returns empty string if no font is available.
func ResolveFont(fonts []FontInfo, family string) string {
	// Exact match first
	for _, f := range fonts {
		if strings.EqualFold(f.Family, family) {
			return f.Path
		}
	}
	// Fallback chain
	for _, fb := range fallbackChain {
		for _, f := range fonts {
			if strings.EqualFold(f.Family, fb) {
				return f.Path
			}
		}
	}
	// Last resort: return first available font
	if len(fonts) > 0 {
		return fonts[0].Path
	}
	return ""
}

// DefaultFontFamily returns the first available font family from the fallback chain,
// or "Impact" if none are detected.
func DefaultFontFamily(fonts []FontInfo) string {
	for _, fb := range fallbackChain {
		for _, f := range fonts {
			if strings.EqualFold(f.Family, fb) {
				return f.Family
			}
		}
	}
	return "Impact"
}
