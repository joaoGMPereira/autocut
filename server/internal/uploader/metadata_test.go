package uploader_test

import (
	"strings"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/uploader"
)

func TestBuildMetadata_MadeForKidsAlwaysFalse(t *testing.T) {
	clip := database.PipelineClip{
		Title:       "Clip title",
		Description: "desc",
		Tags:        "tag1,tag2",
	}
	cfg := database.ChannelConfig{
		ChannelID:         1,
		MadeForKids:       true, // even when channel config says true
		DefaultCategoryID: 22,
		DefaultTags:       "extra",
	}
	meta := uploader.BuildMetadata(clip, cfg)
	if meta.MadeForKids {
		t.Fatal("BuildMetadata must always return MadeForKids=false")
	}
	if meta.Title != "Clip title" {
		t.Fatalf("expected title %q, got %q", "Clip title", meta.Title)
	}
	if !containsTag(meta.Tags, "tag1") {
		t.Fatal("clip tags not included")
	}
	if !containsTag(meta.Tags, "extra") {
		t.Fatal("channel default tags not included")
	}
	if meta.CategoryID != "22" {
		t.Fatalf("expected CategoryID=22, got %q", meta.CategoryID)
	}
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if strings.TrimSpace(t) == tag {
			return true
		}
	}
	return false
}
