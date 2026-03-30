package configurator

import (
	"os"
	"path/filepath"
)

// AutoCutDir holds all directory paths used by AutoCut.
type AutoCutDir struct {
	Root          string // ~/.autocut/
	BinDir        string // ~/.autocut/bin/
	ModelsDir     string // ~/.autocut/models/
	WhisperDir    string // ~/.autocut/models/whisper/
	TokensDir     string // ~/.autocut/tokens/
	CacheDir      string // ~/.autocut/cache/
	ThumbCacheDir string // ~/.autocut/cache/thumbnails/
	TransCacheDir string // ~/.autocut/cache/transcripts/
	DownloadsDir  string // ~/.autocut/downloads/
	ThumbnailsDir string // ~/.autocut/thumbnails/
}

// NewAutoCutDir builds an AutoCutDir rooted at ~/.autocut/.
func NewAutoCutDir() (*AutoCutDir, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return newAutoCutDirFromRoot(filepath.Join(home, ".autocut")), nil
}

// NewAutoCutDirFromRoot creates an AutoCutDir with a custom root path.
// Useful when config specifies a DataDir override.
func NewAutoCutDirFromRoot(root string) *AutoCutDir {
	return newAutoCutDirFromRoot(root)
}

// newAutoCutDirFromRoot is the internal constructor that accepts an explicit root.
// Used by tests to inject a temporary directory without touching HOME.
func newAutoCutDirFromRoot(root string) *AutoCutDir {
	return &AutoCutDir{
		Root:          root,
		BinDir:        filepath.Join(root, "bin"),
		ModelsDir:     filepath.Join(root, "models"),
		WhisperDir:    filepath.Join(root, "models", "whisper"),
		TokensDir:     filepath.Join(root, "tokens"),
		CacheDir:      filepath.Join(root, "cache"),
		ThumbCacheDir: filepath.Join(root, "cache", "thumbnails"),
		TransCacheDir: filepath.Join(root, "cache", "transcripts"),
		DownloadsDir:  filepath.Join(root, "downloads"),
		ThumbnailsDir: filepath.Join(root, "thumbnails"),
	}
}

// Ensure creates every sub-directory (permission 0755) if it does not exist.
func (d *AutoCutDir) Ensure() error {
	dirs := []string{
		d.Root,
		d.BinDir,
		d.ModelsDir,
		d.WhisperDir,
		d.TokensDir,
		d.CacheDir,
		d.ThumbCacheDir,
		d.TransCacheDir,
		d.DownloadsDir,
		d.ThumbnailsDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// BinPath returns the full path for a binary inside BinDir.
func (d *AutoCutDir) BinPath(tool string) string {
	return filepath.Join(d.BinDir, tool)
}

// WhisperModelPath returns the full path for a Whisper model file.
func (d *AutoCutDir) WhisperModelPath(model string) string {
	return filepath.Join(d.WhisperDir, model)
}
