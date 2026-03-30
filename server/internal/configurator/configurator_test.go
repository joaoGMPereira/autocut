package configurator

import (
	"context"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// mockValidator — in-package test double
// ---------------------------------------------------------------------------

type mockValidator struct {
	name      string
	installed bool
	required  bool
}

func (m *mockValidator) Name() string      { return m.name }
func (m *mockValidator) IsInstalled() bool { return m.installed }
func (m *mockValidator) ResolvedPath() string {
	if m.installed {
		return "/mock/path/" + m.name
	}
	return ""
}
func (m *mockValidator) Install(_ context.Context, _ chan<- string) error { return nil }
func (m *mockValidator) Instructions() string                             { return "mock instructions" }
func (m *mockValidator) Status() ToolStatus {
	return ToolStatus{
		Name:      m.name,
		Installed: m.installed,
		Required:  m.required,
		Path:      m.ResolvedPath(),
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func testConfiguratorDir(t *testing.T) *AutoCutDir {
	t.Helper()
	return newAutoCutDirFromRoot(filepath.Join(t.TempDir(), ".autocut"))
}

func allMocksNotInstalled() []ToolValidator {
	return []ToolValidator{
		&mockValidator{name: "yt-dlp", installed: false, required: true},
		&mockValidator{name: "TwitchDownloaderCLI", installed: false, required: false},
		&mockValidator{name: "ffmpeg", installed: false, required: true},
		&mockValidator{name: "whisper", installed: false, required: false},
		&mockValidator{name: "ollama", installed: false, required: false},
		&mockValidator{name: "convert", installed: false, required: false},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestStatus(t *testing.T) {
	c := New(testConfiguratorDir(t))
	statuses := c.Status()
	if len(statuses) < 6 {
		t.Errorf("Status() len = %d, want >= 6", len(statuses))
	}
}

func TestMissing(t *testing.T) {
	dir := testConfiguratorDir(t)
	c := newWithValidators(dir, allMocksNotInstalled())
	missing := c.Missing()
	if len(missing) != 6 {
		t.Errorf("Missing() len = %d, want 6", len(missing))
	}
	for _, s := range missing {
		if s.Installed {
			t.Errorf("tool %q reported as installed in Missing() result", s.Name)
		}
	}
}

func TestGetValidator(t *testing.T) {
	dir := testConfiguratorDir(t)
	c := newWithValidators(dir, allMocksNotInstalled())

	v, ok := c.Get("ffmpeg")
	if !ok {
		t.Fatal("Get('ffmpeg') not found")
	}
	if v.Name() != "ffmpeg" {
		t.Errorf("Get('ffmpeg').Name() = %q, want 'ffmpeg'", v.Name())
	}

	_, ok = c.Get("noop")
	if ok {
		t.Error("Get('noop') should return false")
	}
}

func TestAllInstalled(t *testing.T) {
	t.Run("true when all required installed", func(t *testing.T) {
		cfg := newWithValidators(&AutoCutDir{}, []ToolValidator{
			&mockValidator{name: "req1", installed: true, required: true},
			&mockValidator{name: "opt1", installed: false, required: false},
		})
		if !cfg.AllInstalled() {
			t.Error("expected AllInstalled() == true")
		}
	})
	t.Run("false when required missing", func(t *testing.T) {
		cfg := newWithValidators(&AutoCutDir{}, []ToolValidator{
			&mockValidator{name: "req1", installed: false, required: true},
		})
		if cfg.AllInstalled() {
			t.Error("expected AllInstalled() == false")
		}
	})
}
