package uploader_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/uploader"
)

func TestSaveToQueue_CreatesFilesAndMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	qs := uploader.NewQueueStorage(tmpDir)

	// Create dummy source files
	srcVideo := filepath.Join(tmpDir, "clip.mp4")
	srcThumb := filepath.Join(tmpDir, "thumb.jpg")
	if err := os.WriteFile(srcVideo, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcThumb, []byte("thumb"), 0o644); err != nil {
		t.Fatal(err)
	}

	meta := uploader.VideoMetadata{
		ChannelID:   3,
		Title:       "Test clip",
		Description: "desc",
		Tags:        []string{"tag1"},
		CategoryID:  "22",
		Privacy:     "private",
		MadeForKids: false,
		Language:    "pt-BR",
	}

	qVideo, qThumb, err := qs.SaveToQueue(srcVideo, srcThumb, meta)
	if err != nil {
		t.Fatalf("SaveToQueue failed: %v", err)
	}

	// video + thumbnail must exist at returned paths
	if _, err := os.Stat(qVideo); err != nil {
		t.Fatalf("queued video not found: %v", err)
	}
	if _, err := os.Stat(qThumb); err != nil {
		t.Fatalf("queued thumbnail not found: %v", err)
	}

	// metadata.json must exist in same dir as video
	metaPath := filepath.Join(filepath.Dir(qVideo), "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("metadata.json not found: %v", err)
	}
	var got uploader.VideoMetadata
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse metadata.json: %v", err)
	}
	if got.Title != "Test clip" {
		t.Fatalf("metadata title mismatch: %q", got.Title)
	}
	if got.MadeForKids {
		t.Fatal("metadata.json must have made_for_kids=false")
	}
}

func TestSaveToQueue_NoThumbnail(t *testing.T) {
	tmpDir := t.TempDir()
	qs := uploader.NewQueueStorage(tmpDir)

	srcVideo := filepath.Join(tmpDir, "clip.mp4")
	if err := os.WriteFile(srcVideo, []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}

	qVideo, qThumb, err := qs.SaveToQueue(srcVideo, "", uploader.VideoMetadata{Title: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(qVideo); err != nil {
		t.Fatal("video must be saved even without thumbnail")
	}
	if qThumb != "" {
		t.Fatalf("qThumb should be empty when no thumbnail provided, got %q", qThumb)
	}
}
