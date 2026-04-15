package database

import "database/sql"

// Kotlin: data class Channel in ChannelRepository.kt
type Channel struct {
	ID                  int64
	Name                string
	ChannelID           string
	ChannelTitle        string
	AvatarURL           string
	AccessToken         string
	RefreshToken        string
	ExpiresAt           int64
	OAuthClientSecretID sql.NullInt64
	IsFavorite          bool
	CreatedAt           int64
	UpdatedAt           int64
}

// Kotlin: VideosTable in Tables.kt (no dedicated data class)
type Video struct {
	ID        int64
	ChannelID sql.NullInt64
	URL       string
	Platform  string
	Title     string
	FilePath  string
	Duration  int64
	Status    string
	CreatedAt int64
}

// Kotlin: ClipsTable in Tables.kt (no dedicated data class)
type Clip struct {
	ID            int64
	VideoID       int64
	StartTime     float64
	EndTime       float64
	FilePath      string
	Title         string
	Description   string
	Tags          string
	ThumbnailPath string
	Status        string
	CreatedAt     int64
}

// Kotlin: data class Upload in UploadRepository.kt
type Upload struct {
	ID                 int64
	ClipID             int64
	ChannelID          int64
	YoutubeID          string
	YoutubeURL         string
	Status             string
	ScheduledAt        sql.NullInt64
	UploadedAt         sql.NullInt64
	Error              string
	VideoType          string
	LocalVideoPath     sql.NullString
	LocalThumbnailPath sql.NullString
	MetadataJSON       string
	UploadConfigJSON   string
	OriginalVideoName  string
	SourceVideoURL     string
	SourceClipURL      string
	ShortsGenerated    bool
	ShortsGeneratedAt  sql.NullInt64
	CreatedAt          int64
}

// Kotlin: PipelineRunsTable in Tables.kt (no dedicated data class)
// PipelineRun maps to the v10 pipeline_runs schema.
type PipelineRun struct {
	ID             int64         `json:"id"`
	ChannelID      sql.NullInt64 `json:"channel_id"`
	URL            string        `json:"url"`
	Mode           string        `json:"mode"`
	State          string        `json:"state"`        // WAITING_URL | EXECUTING | WAITING_MODE | DONE | ERROR | CANCELLED
	ActivePhase    string        `json:"active_phase"` // download | transcribe | …
	Error          string        `json:"error"`
	ModeConfigJSON string        `json:"-"`
	VideoPath      string        `json:"video_path"`
	VideoTitle     string        `json:"video_title"`
	DurationSec    int64         `json:"duration_sec"`
	TranscriptPath string        `json:"transcript_path"`
	StartedAt      int64         `json:"started_at"`
	FinishedAt     sql.NullInt64 `json:"finished_at"`
}

// PipelineRunStep tracks individual step execution within a pipeline run.
type PipelineRunStep struct {
	ID         int64         `json:"id"`
	RunID      int64         `json:"run_id"`
	Step       string        `json:"step"`
	Status     string        `json:"status"`
	JobID      string        `json:"job_id"`
	Error      string        `json:"error"`
	StartedAt  sql.NullInt64 `json:"started_at"`
	FinishedAt sql.NullInt64 `json:"finished_at"`
}

// Kotlin: data class ThumbnailBackground in ThumbnailBackgroundRepository.kt
type ThumbnailBackground struct {
	ID        int64
	ChannelID int64
	Name      string
	FilePath  string
	IsDefault bool
	CreatedAt int64
}

// Kotlin: ChannelConfigTable in Tables.kt + delegates
type ChannelConfig struct {
	ID        int64 `json:"id"`
	ChannelID int64 `json:"channel_id"`

	// Gradient colors (thumbnail generation)
	GradientColorStart string `json:"gradient_color_start"`
	GradientColorEnd   string `json:"gradient_color_end"`

	// Thumbnail text styling
	ThumbnailFontFamily  string `json:"thumbnail_font_family"`
	ThumbnailTextColor1  string `json:"thumbnail_text_color1"`
	ThumbnailTextColor2  string `json:"thumbnail_text_color2"`
	ThumbnailTextColor3  string `json:"thumbnail_text_color3"`

	// YouTube metadata defaults
	MadeForKids              bool           `json:"made_for_kids"`
	DefaultCategoryID        int            `json:"default_category_id"`
	DefaultPlaylistIDCortes  sql.NullString `json:"default_playlist_id_cortes"`
	DefaultPlaylistIDShorts  sql.NullString `json:"default_playlist_id_shorts"`
	DefaultTags              string         `json:"default_tags"`
	ShortsTitleHashtags      string         `json:"shorts_title_hashtags"`
	VideoDescriptionHashtags string         `json:"video_description_hashtags"`

	// Anti-duplicate configuration
	AntiDuplicateEnabled bool   `json:"anti_duplicate_enabled"`
	AntiDuplicateMode    string `json:"anti_duplicate_mode"`

	// Speed configuration
	SpeedEnabled bool    `json:"speed_enabled"`
	SpeedFactor  float64 `json:"speed_factor"`

	// Visual configuration
	VisualCropEnabled         bool    `json:"visual_crop_enabled"`
	VisualCropPercent         float64 `json:"visual_crop_percent"`
	VisualZoomEnabled         bool    `json:"visual_zoom_enabled"`
	VisualZoomAmount          float64 `json:"visual_zoom_amount"`
	VisualColorGradingEnabled bool    `json:"visual_color_grading_enabled"`
	VisualBrightness          float64 `json:"visual_brightness"`
	VisualSaturation          float64 `json:"visual_saturation"`
	VisualContrast            float64 `json:"visual_contrast"`

	// Branding configuration
	BrandingLogoEnabled           bool           `json:"branding_logo_enabled"`
	BrandingLogoPath              sql.NullString `json:"branding_logo_path"`
	BrandingLogoPosition          string         `json:"branding_logo_position"`
	BrandingLogoOpacity           float64        `json:"branding_logo_opacity"`
	BrandingLogoScale             float64        `json:"branding_logo_scale"`
	BrandingIntroEnabled          bool           `json:"branding_intro_enabled"`
	BrandingIntroPath             sql.NullString `json:"branding_intro_path"`
	BrandingGenerateIntroFromLogo bool           `json:"branding_generate_intro_from_logo"`
	BrandingIntroDuration         float64        `json:"branding_intro_duration"`
	BrandingOutroEnabled          bool           `json:"branding_outro_enabled"`
	BrandingOutroPath             sql.NullString `json:"branding_outro_path"`

	// Audio configuration
	AudioMusicEnabled bool           `json:"audio_music_enabled"`
	AudioMusicPath    sql.NullString `json:"audio_music_path"`
	AudioMusicVolume  float64        `json:"audio_music_volume"`
	AudioRandomMusic  bool           `json:"audio_random_music"`
	AudioMusicFolder  string         `json:"audio_music_folder"`

	// Pinned comment
	PinnedCommentTemplate string `json:"pinned_comment_template"`

	// Preview/caption configuration
	PreviewCaptionsEnabled        bool   `json:"preview_captions_enabled"`
	PreviewCaptionStyle           string `json:"preview_caption_style"`
	PreviewCaptionTextStyleJSON   string `json:"preview_caption_text_style_json"`
	PreviewVideoOverlayConfigJSON string `json:"preview_video_overlay_config_json"`
	PreviewTextOverlayConfigJSON  string `json:"preview_text_overlay_config_json"`

	// Max highlights
	MaxHighlights sql.NullInt64 `json:"max_highlights"`

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// MusicBlacklistEntry represents a per-channel music pattern to exclude.
// FP-009: music_blacklist table.
type MusicBlacklistEntry struct {
	ID        int64  `json:"id"`
	ChannelID int64  `json:"channel_id"`
	Pattern   string `json:"pattern"`
	CreatedAt string `json:"created_at"`
}

// Kotlin: AppSettingsTable in Tables.kt
type AppSetting struct {
	ID        int64  `json:"id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt int64  `json:"updated_at"`
}

// Kotlin: data class OAuthClientSecret in OAuthClientSecretRepository.kt
type OAuthClientSecret struct {
	ID           int64
	Name         string
	ClientID     string
	ClientSecret string
	ProjectID    string
	IsDefault    bool
	CreatedAt    int64
	UpdatedAt    int64
}

// Kotlin: data class ShortRecord in ShortsRepository.kt
type Short struct {
	ID                  int64
	SourceVideoPath     string
	SegmentJSON         string
	FilePath            string
	ThumbnailPath       sql.NullString
	CustomThumbnailPath sql.NullString
	Title               string
	ThumbnailTitle      sql.NullString
	Description         string
	Tags                string
	Duration            float64
	ChannelID           sql.NullInt64
	YoutubeID           sql.NullString
	YoutubeURL          sql.NullString
	Status              string
	ScheduledAt         sql.NullInt64
	UploadedAt          sql.NullInt64
	Error               string
	CreatedAt           int64
}

// Kotlin: data class QuotaUsage in QuotaRepository.kt
type QuotaUsage struct {
	ID                   int64
	OAuthClientSecretID  int64
	Date                 string
	UnitsUsed            int
	UploadCount          int
	ThumbnailCount       int
	OtherAPICalls        int
	CreatedAt            int64
	UpdatedAt            int64
}

// URLHistoryEntry represents a persisted URL in the url_history table (migration v11).
type URLHistoryEntry struct {
	ID         int64   `json:"id"`
	URL        string  `json:"url"`
	VideoTitle *string `json:"video_title"`
	LastUsedAt int64   `json:"last_used_at"`
	UseCount   int     `json:"use_count"`
}

// Quota constants from Kotlin QuotaUsage companion object
const (
	QuotaDailyLimit    = 10_000
	QuotaUploadCost    = 1_600
	QuotaThumbnailCost = 50
	QuotaCommentCost   = 50
	QuotaListCost      = 1
)
