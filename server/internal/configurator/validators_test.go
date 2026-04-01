package configurator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	// This test only makes sense on a system where ffmpeg is NOT installed at
	// all (not in PATH and not in any well-known directory). Skip if ffmpeg
	// is available, since Install() will succeed by copying or downloading it.
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		t.Skip("ffmpeg found in PATH; Install() will succeed — skipping error test")
	}
	if discoverLocalBinary("ffmpeg") != "" {
		t.Skip("ffmpeg found in well-known path; Install() will succeed — skipping error test")
	}
	// On macOS the download would actually succeed (evermeet.cx), so skip there too.
	if runtime.GOOS == "darwin" {
		t.Skip("Install() downloads from evermeet.cx on macOS — skipping error test")
	}
	v := NewFFmpegValidator(testDir(t))
	err := v.Install(context.Background(), make(chan<- string, 1))
	if err == nil {
		t.Error("expected Install() to return an error for FFmpeg")
	}
}

func TestFFmpegAutoSymlink(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not in PATH, skipping auto-symlink test")
	}
	dir := testDir(t)
	if err := dir.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	v := NewFFmpegValidator(dir)
	v.Status() // should create symlink as side-effect
	if !statExists(dir.BinPath("ffmpeg")) {
		t.Error("expected ~/.autocut/bin/ffmpeg to exist after Status()")
	}
	// ResolvedPath must now point to the bin/ symlink, not the system path
	got := v.ResolvedPath()
	if got != dir.BinPath("ffmpeg") {
		t.Errorf("ResolvedPath() = %q, want %q", got, dir.BinPath("ffmpeg"))
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

// ---------------------------------------------------------------------------
// discoverLocalBinary
// ---------------------------------------------------------------------------

func TestDiscoverLocalBinary(t *testing.T) {
	got := discoverLocalBinary("ffmpeg")
	if got != "" && !statExists(got) {
		t.Errorf("discoverLocalBinary returned %q but file does not exist", got)
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
