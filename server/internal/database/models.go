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
type PipelineRun struct {
	ID          int64
	ChannelID   sql.NullInt64
	URL         string
	Mode        string
	Status      string
	Progress    int
	CurrentStep string
	Error       string
	StartedAt   int64
	FinishedAt  sql.NullInt64
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
	ID        int64
	ChannelID int64

	// Gradient colors (thumbnail generation)
	GradientColorStart string
	GradientColorEnd   string

	// Thumbnail text styling
	ThumbnailFontFamily  string
	ThumbnailTextColor1  string
	ThumbnailTextColor2  string
	ThumbnailTextColor3  string

	// YouTube metadata defaults
	MadeForKids              bool
	DefaultCategoryID        int
	DefaultPlaylistIDCortes  sql.NullString
	DefaultPlaylistIDShorts  sql.NullString
	DefaultTags              string
	ShortsTitleHashtags      string
	VideoDescriptionHashtags string

	// Anti-duplicate configuration
	AntiDuplicateEnabled bool
	AntiDuplicateMode    string

	// Speed configuration
	SpeedEnabled bool
	SpeedFactor  float64

	// Visual configuration
	VisualCropEnabled         bool
	VisualCropPercent         float64
	VisualZoomEnabled         bool
	VisualZoomAmount          float64
	VisualColorGradingEnabled bool
	VisualBrightness          float64
	VisualSaturation          float64
	VisualContrast            float64

	// Branding configuration
	BrandingLogoEnabled            bool
	BrandingLogoPath               sql.NullString
	BrandingLogoPosition           string
	BrandingLogoOpacity            float64
	BrandingLogoScale              float64
	BrandingIntroEnabled           bool
	BrandingIntroPath              sql.NullString
	BrandingGenerateIntroFromLogo  bool
	BrandingIntroDuration          float64
	BrandingOutroEnabled           bool
	BrandingOutroPath              sql.NullString

	// Audio configuration
	AudioMusicEnabled bool
	AudioMusicPath    sql.NullString
	AudioMusicVolume  float64
	AudioRandomMusic  bool
	AudioMusicFolder  string

	// Pinned comment
	PinnedCommentTemplate string

	// Preview/caption configuration
	PreviewCaptionsEnabled         bool
	PreviewCaptionStyle            string
	PreviewCaptionTextStyleJSON    string
	PreviewVideoOverlayConfigJSON  string
	PreviewTextOverlayConfigJSON   string

	// Max highlights
	MaxHighlights sql.NullInt64

	CreatedAt int64
	UpdatedAt int64
}

// Kotlin: AppSettingsTable in Tables.kt
type AppSetting struct {
	ID        int64
	Key       string
	Value     string
	UpdatedAt int64
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

// Quota constants from Kotlin QuotaUsage companion object
const (
	QuotaDailyLimit    = 10_000
	QuotaUploadCost    = 1_600
	QuotaThumbnailCost = 50
	QuotaCommentCost   = 50
	QuotaListCost      = 1
)
