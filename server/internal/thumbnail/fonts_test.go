package thumbnail

import (
	"testing"
)

func TestFontDetectorList(t *testing.T) {
	d := NewFontDetector()
	fonts, err := d.List()
	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	// An empty result is valid on CI runners that have no fonts installed.
	// We only verify that the returned slice contains well-formed entries.
	for _, f := range fonts {
		if f.Name == "" {
			t.Errorf("FontInfo with empty Name: %+v", f)
		}
		if f.Path == "" {
			t.Errorf("FontInfo with empty Path: %+v", f)
		}
	}
	t.Logf("List() returned %d font(s)", len(fonts))
}

func TestFontDetectorFind(t *testing.T) {
	d := NewFontDetector()
	fonts, err := d.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(fonts) == 0 {
		t.Skip("no fonts available on this system, skipping Find test")
	}

	// Pick the first discovered font and look it up by name.
	target := fonts[0].Name
	found, err := d.Find(target)
	if err != nil {
		t.Fatalf("Find(%q) error: %v", target, err)
	}
	if found == nil {
		t.Fatalf("Find(%q) returned nil without error", target)
	}
	if found.Name != target {
		t.Errorf("Find(%q) returned Name=%q, want %q", target, found.Name, target)
	}
}

func TestFontDetectorFindNotFound(t *testing.T) {
	d := NewFontDetector()
	_, err := d.Find("__definitely_does_not_exist_xyz__")
	if err == nil {
		t.Error("Find() with non-existent name should return an error")
	}
}

func TestSystemDefault(t *testing.T) {
	d := NewFontDetector()
	name := d.SystemDefault()
	if name == "" {
		t.Error("SystemDefault() returned empty string")
	}
	t.Logf("SystemDefault() = %q", name)
}
