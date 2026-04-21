package uploader_test

import (
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/uploader"
)

// TestVideoMetadata_ZeroValueSafe verifies that a zero-initialized VideoMetadata
// has MadeForKids=false, which is the invariant enforced throughout the codebase.
func TestVideoMetadata_ZeroValueSafe(t *testing.T) {
	var meta uploader.VideoMetadata
	if meta.MadeForKids {
		t.Fatal("zero-value VideoMetadata must have MadeForKids=false")
	}
}

// TestVideoMetadata_Fields verifies all expected fields compile and are accessible.
func TestVideoMetadata_Fields(t *testing.T) {
	meta := uploader.VideoMetadata{
		ChannelID:   3,
		Title:       "Test",
		Description: "Desc",
		Tags:        []string{"a", "b"},
		CategoryID:  "22",
		Privacy:     "private",
		MadeForKids: false,
		Language:    "pt-BR",
	}
	if meta.Title != "Test" || len(meta.Tags) != 2 || meta.MadeForKids {
		t.Fatal("field access failed")
	}
}
