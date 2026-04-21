package uploader

// VideoMetadata holds all YouTube upload metadata fields.
// MadeForKids is always false — never read from channel config.
type VideoMetadata struct {
	ChannelID   int64    `json:"channel_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	CategoryID  string   `json:"category_id"`
	Privacy     string   `json:"privacy"`
	MadeForKids bool     `json:"made_for_kids"`
	Language    string   `json:"language"`
}
