package configurator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewAutoCutDir(t *testing.T) {
	d, err := NewAutoCutDir()
	if err != nil {
		t.Fatalf("NewAutoCutDir() error = %v", err)
	}
	if !strings.HasSuffix(d.Root, ".autocut") {
		t.Errorf("Root = %q, want suffix '.autocut'", d.Root)
	}
}

func TestEnsureCreatesAllDirs(t *testing.T) {
	tmp := t.TempDir()
	d := newAutoCutDirFromRoot(filepath.Join(tmp, ".autocut"))
	if err := d.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

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
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory to exist: %s", dir)
		}
	}
}

func TestBinPath(t *testing.T) {
	tmp := t.TempDir()
	d := newAutoCutDirFromRoot(filepath.Join(tmp, ".autocut"))
	want := filepath.Join(d.BinDir, "yt-dlp")
	got := d.BinPath("yt-dlp")
	if got != want {
		t.Errorf("BinPath('yt-dlp') = %q, want %q", got, want)
	}
}

func TestWhisperModelPath(t *testing.T) {
	tmp := t.TempDir()
	d := newAutoCutDirFromRoot(filepath.Join(tmp, ".autocut"))
	want := filepath.Join(d.WhisperDir, "ggml-base.bin")
	got := d.WhisperModelPath("ggml-base.bin")
	if got != want {
		t.Errorf("WhisperModelPath('ggml-base.bin') = %q, want %q", got, want)
	}
}
