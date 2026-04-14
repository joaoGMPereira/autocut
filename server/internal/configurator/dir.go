package configurator

import (
	"fmt"
	"os"
	"path/filepath"
)

// Ensure creates all AutoCutDir subdirectories.
func (d *AutoCutDir) Ensure() error {
	dirs := []string{
		d.BinDir,
		d.ModelsDir,
		d.WhisperDir,
		d.TokensDir,
		d.CacheDir,
		d.DownloadsDir,
		d.ThumbnailsDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return nil
}

// BinPath returns the absolute path of a binary in BinDir.
func (d *AutoCutDir) BinPath(name string) string {
	return filepath.Join(d.BinDir, name)
}

// WhisperModelPath returns the absolute path of a Whisper model file.
func (d *AutoCutDir) WhisperModelPath(filename string) string {
	return filepath.Join(d.WhisperDir, filename)
}
