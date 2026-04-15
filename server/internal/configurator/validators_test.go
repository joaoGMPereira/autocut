package configurator

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// US2 — TwitchDownloaderCLI required flag
// ---------------------------------------------------------------------------

func TestTwitchValidatorRequired(t *testing.T) {
	dir := testConfiguratorDir(t)
	v := NewTwitchValidator(dir, nil)
	if !v.Status().Required {
		t.Fatalf("expected TwitchValidator.Required = true, got false")
	}
}

// ---------------------------------------------------------------------------
// US4 — Whisper model file detection
// ---------------------------------------------------------------------------

func TestHasWhisperModel_NoFiles(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "whisper-cli")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if hasWhisperModel(binaryPath, tmp, "") {
		t.Fatal("expected hasWhisperModel to return false when no .bin files exist")
	}
}

func TestHasWhisperModel_FileInBinaryDir(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "whisper-cli")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(tmp, "ggml-base.bin")
	if err := os.WriteFile(modelPath, []byte("fake model"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasWhisperModel(binaryPath, tmp, "") {
		t.Fatal("expected hasWhisperModel to return true when ggml-*.bin exists in binary dir")
	}
}

func TestHasWhisperModel_FileInModelsSubdir(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "whisper-cli")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	modelsDir := filepath.Join(tmp, "models")
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelsDir, "ggml-small.bin"), []byte("fake model"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasWhisperModel(binaryPath, tmp, "") {
		t.Fatal("expected hasWhisperModel to return true when ggml-*.bin exists in models subdir")
	}
}

func TestWhisperValidatorNoModel(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "whisper-cli")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	v := &WhisperValidator{
		baseValidator: baseValidator{
			name:     "whisper-cli",
			path:     binaryPath,
			required: true,
			source:   "autocut_bin",
			binDir:   tmp,
		},
	}
	s := v.Status()
	if s.Installed {
		t.Fatalf("expected Installed=false when no model file present, got true")
	}
	if s.Source != "no_model" {
		t.Fatalf("expected Source=%q, got %q", "no_model", s.Source)
	}
}

func TestWhisperValidatorWithModel(t *testing.T) {
	tmp := t.TempDir()
	binaryPath := filepath.Join(tmp, "whisper-cli")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "ggml-base.bin"), []byte("fake model"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := &WhisperValidator{
		baseValidator: baseValidator{
			name:     "whisper-cli",
			path:     binaryPath,
			required: true,
			source:   "autocut_bin",
			binDir:   tmp,
		},
	}
	s := v.Status()
	if !s.Installed {
		t.Fatalf("expected Installed=true when ggml-base.bin exists in binary dir, got false")
	}
}
