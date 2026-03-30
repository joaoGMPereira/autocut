package transcript

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeTestTranscript returns a small Transcript for round-trip testing.
func makeTestTranscript() *Transcript {
	return &Transcript{
		Segments: []Segment{
			{Start: 0, End: 2 * time.Second, Text: "hello", Confidence: 0.9},
			{Start: 3 * time.Second, End: 5 * time.Second, Text: "world", Confidence: 0.85},
		},
		Language: "en",
		Duration: 5 * time.Second,
	}
}

func TestHashDeterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.mp4")
	if err := os.WriteFile(path, []byte("fake video data for hashing"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := NewCache(dir)

	h1, err := cache.Hash(path)
	if err != nil {
		t.Fatalf("first Hash call failed: %v", err)
	}
	h2, err := cache.Hash(path)
	if err != nil {
		t.Fatalf("second Hash call failed: %v", err)
	}

	if h1 != h2 {
		t.Errorf("Hash is not deterministic: %q != %q", h1, h2)
	}
	if h1 == "" {
		t.Error("Hash returned empty string")
	}
}

func TestCacheMiss(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	tr, ok := cache.Get("nonexistent-hash-abc123")
	if ok {
		t.Errorf("expected cache miss, got hit with transcript %+v", tr)
	}
	if tr != nil {
		t.Errorf("expected nil transcript on miss, got %+v", tr)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	original := makeTestTranscript()
	hash := "deadbeef1234"

	if err := cache.Put(hash, original); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	got, ok := cache.Get(hash)
	if !ok {
		t.Fatal("expected cache hit after Put, got miss")
	}

	if got.Language != original.Language {
		t.Errorf("Language: want %q, got %q", original.Language, got.Language)
	}
	if got.Duration != original.Duration {
		t.Errorf("Duration: want %v, got %v", original.Duration, got.Duration)
	}
	if len(got.Segments) != len(original.Segments) {
		t.Fatalf("Segments len: want %d, got %d", len(original.Segments), len(got.Segments))
	}
	for i, seg := range original.Segments {
		g := got.Segments[i]
		if g.Start != seg.Start || g.End != seg.End || g.Text != seg.Text {
			t.Errorf("Segments[%d]: want %+v, got %+v", i, seg, g)
		}
	}
}

func TestHashDifferentFiles(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(dir)

	p1 := filepath.Join(dir, "a.mp4")
	p2 := filepath.Join(dir, "b.mp4")
	if err := os.WriteFile(p1, []byte("content A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, []byte("content B"), 0o644); err != nil {
		t.Fatal(err)
	}

	h1, _ := cache.Hash(p1)
	h2, _ := cache.Hash(p2)
	if h1 == h2 {
		t.Error("different files produced the same hash")
	}
}
