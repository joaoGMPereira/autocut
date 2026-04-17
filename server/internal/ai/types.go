package ai

// ClipMetadata holds AI-generated metadata for a single video clip.
type ClipMetadata struct {
	ClipID        int64    `json:"clip_id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	ThumbnailText string   `json:"thumbnail_text"`
	CategoryID    int      `json:"category_id"`
}

// GenerateRequest holds the parameters for a Claude CLI invocation.
type GenerateRequest struct {
	Prompt       string
	SystemPrompt string
	Model        string // "haiku", "sonnet", "opus"
}
