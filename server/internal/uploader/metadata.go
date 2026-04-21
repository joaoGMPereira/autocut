package uploader

import (
	"fmt"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// BuildMetadata merges clip + channel config into a VideoMetadata value.
// MadeForKids is always false regardless of channel config.
func BuildMetadata(clip database.PipelineClip, cfg database.ChannelConfig) VideoMetadata {
	tags := splitTags(clip.Tags)
	if cfg.DefaultTags != "" {
		tags = append(tags, splitTags(cfg.DefaultTags)...)
	}
	return VideoMetadata{
		ChannelID:   cfg.ChannelID,
		Title:       clip.Title,
		Description: clip.Description,
		Tags:        tags,
		CategoryID:  fmt.Sprintf("%d", cfg.DefaultCategoryID),
		Privacy:     "private",
		MadeForKids: false, // hardcoded — never from channel config
		Language:    "pt-BR",
	}
}

// splitTags splits a comma-separated tag string, trimming whitespace.
func splitTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
