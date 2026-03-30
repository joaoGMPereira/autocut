package configurator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func testDir(t *testing.T) *AutoCutDir {
	t.Helper()
	return newAutoCutDirFromRoot(filepath.Join(t.TempDir(), ".autocut"))
}

// ---------------------------------------------------------------------------
// FFmpegValidator
// ---------------------------------------------------------------------------

func TestFFmpegInstructions(t *testing.T) {
	v := NewFFmpegValidator(testDir(t))
	if instr := v.Instructions(); instr == "" {
		t.Error("Instructions() returned empty string")
	}
}

func TestFFmpegInstallError(t *testing.T) {
	v := NewFFmpegValidator(testDir(t))
	err := v.Install(context.Background(), make(chan<- string, 1))
	if err == nil {
		t.Error("expected Install() to return an error for FFmpeg")
	}
}

// ---------------------------------------------------------------------------
// OllamaValidator
// ---------------------------------------------------------------------------

func TestOllamaNotRunning(t *testing.T) {
	// Use a port that is guaranteed to be closed.
	v := newOllamaValidatorWithURL(testDir(t), "http://localhost:19999/")
	if v.IsInstalled() {
		t.Error("expected IsInstalled() = false when server is not reachable")
	}
}

// ---------------------------------------------------------------------------
// YtDlpValidator — ResolvedPath fallback
// ---------------------------------------------------------------------------

func TestYtDlpResolvedPathFallback(t *testing.T) {
	dir := testDir(t)
	// Ensure BinDir exists and place a fake yt-dlp binary there.
	if err := dir.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	fakeBin := dir.BinPath("yt-dlp")
	f, err := os.Create(fakeBin)
	if err != nil {
		t.Fatalf("create fake binary: %v", err)
	}
	f.Close()

	v := NewYtDlpValidator(dir)
	// We cannot control $PATH here, but we can at least verify that when the
	// file exists in BinDir the validator is considered installed and returns
	// the BinDir path (or an overriding PATH path).
	if !v.IsInstalled() {
		t.Error("expected IsInstalled() = true when binary exists in BinDir")
	}
	got := v.ResolvedPath()
	if got == "" {
		t.Error("ResolvedPath() returned empty string despite binary in BinDir")
	}
}
