package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// ChannelConfigRepo provides CRUD for channel_config (1:1 with channels).
// Kotlin ref: ChannelConfigRepository.kt + 4 delegates
type ChannelConfigRepo struct {
	db  *sql.DB
	log *slog.Logger
}

func NewChannelConfigRepo(db *sql.DB, log *slog.Logger) *ChannelConfigRepo {
	return &ChannelConfigRepo{db: db, log: log.With("repo", "channel_config")}
}

const channelConfigCols = `id, channel_id,
	gradient_color_start, gradient_color_end,
	thumbnail_font_family, thumbnail_text_color_1, thumbnail_text_color_2, thumbnail_text_color_3,
	made_for_kids, default_category_id, default_playlist_id_cortes, default_playlist_id_shorts,
	default_tags, shorts_title_hashtags, video_description_hashtags,
	anti_duplicate_enabled, anti_duplicate_mode,
	speed_enabled, speed_factor,
	visual_crop_enabled, visual_crop_percent, visual_zoom_enabled, visual_zoom_amount,
	visual_color_grading_enabled, visual_brightness, visual_saturation, visual_contrast,
	branding_logo_enabled, branding_logo_path, branding_logo_position, branding_logo_opacity, branding_logo_scale,
	branding_intro_enabled, branding_intro_path, branding_generate_intro_from_logo, branding_intro_duration,
	branding_outro_enabled, branding_outro_path,
	audio_music_enabled, audio_music_path, audio_music_volume, audio_random_music, audio_music_folder,
	pinned_comment_template,
	preview_captions_enabled, preview_caption_style, preview_caption_text_style_json,
	preview_video_overlay_config_json, preview_text_overlay_config_json,
	max_highlights,
	created_at, updated_at`

// GetByChannelID returns the config for a channel, creating a default if none exists.
// Kotlin ref: getByChannelId(channelId) — returns default if not found
func (r *ChannelConfigRepo) GetByChannelID(ctx context.Context, channelID int64) (*ChannelConfig, error) {
	cc, err := r.GetByChannelIDOrNil(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if cc != nil {
		return cc, nil
	}
	// Create default config
	if err := r.CreateDefault(ctx, channelID); err != nil {
		return nil, err
	}
	return r.GetByChannelIDOrNil(ctx, channelID)
}

// GetByChannelIDOrNil returns nil if no config exists.
// Kotlin ref: getByChannelIdOrNull(channelId)
func (r *ChannelConfigRepo) GetByChannelIDOrNil(ctx context.Context, channelID int64) (*ChannelConfig, error) {
	row := r.db.QueryRowContext(ctx,
		"SELECT "+channelConfigCols+" FROM channel_config WHERE channel_id = ?", channelID)
	cc, err := scanChannelConfig(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return cc, nil
}

// CreateDefault inserts a config row with all default values.
// Kotlin ref: createDefault(channelId)
func (r *ChannelConfigRepo) CreateDefault(ctx context.Context, channelID int64) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO channel_config (channel_id, created_at, updated_at) VALUES (?, ?, ?)",
		channelID, now, now,
	)
	if err != nil {
		return fmt.Errorf("create default channel_config for channel %d: %w", channelID, err)
	}
	return nil
}

// Update replaces all config fields at once.
// Kotlin ref: updateConfig(chId, config) — consolidates 20+ granular update methods
func (r *ChannelConfigRepo) Update(ctx context.Context, cc *ChannelConfig) error {
	now := time.Now().UnixMilli()
	_, err := r.db.ExecContext(ctx,
		`UPDATE channel_config SET
			gradient_color_start = ?, gradient_color_end = ?,
			thumbnail_font_family = ?, thumbnail_text_color_1 = ?, thumbnail_text_color_2 = ?, thumbnail_text_color_3 = ?,
			made_for_kids = ?, default_category_id = ?, default_playlist_id_cortes = ?, default_playlist_id_shorts = ?,
			default_tags = ?, shorts_title_hashtags = ?, video_description_hashtags = ?,
			anti_duplicate_enabled = ?, anti_duplicate_mode = ?,
			speed_enabled = ?, speed_factor = ?,
			visual_crop_enabled = ?, visual_crop_percent = ?, visual_zoom_enabled = ?, visual_zoom_amount = ?,
			visual_color_grading_enabled = ?, visual_brightness = ?, visual_saturation = ?, visual_contrast = ?,
			branding_logo_enabled = ?, branding_logo_path = ?, branding_logo_position = ?, branding_logo_opacity = ?, branding_logo_scale = ?,
			branding_intro_enabled = ?, branding_intro_path = ?, branding_generate_intro_from_logo = ?, branding_intro_duration = ?,
			branding_outro_enabled = ?, branding_outro_path = ?,
			audio_music_enabled = ?, audio_music_path = ?, audio_music_volume = ?, audio_random_music = ?, audio_music_folder = ?,
			pinned_comment_template = ?,
			preview_captions_enabled = ?, preview_caption_style = ?, preview_caption_text_style_json = ?,
			preview_video_overlay_config_json = ?, preview_text_overlay_config_json = ?,
			max_highlights = ?,
			updated_at = ?
		 WHERE channel_id = ?`,
		cc.GradientColorStart, cc.GradientColorEnd,
		cc.ThumbnailFontFamily, cc.ThumbnailTextColor1, cc.ThumbnailTextColor2, cc.ThumbnailTextColor3,
		cc.MadeForKids, cc.DefaultCategoryID, cc.DefaultPlaylistIDCortes, cc.DefaultPlaylistIDShorts,
		cc.DefaultTags, cc.ShortsTitleHashtags, cc.VideoDescriptionHashtags,
		cc.AntiDuplicateEnabled, cc.AntiDuplicateMode,
		cc.SpeedEnabled, cc.SpeedFactor,
		cc.VisualCropEnabled, cc.VisualCropPercent, cc.VisualZoomEnabled, cc.VisualZoomAmount,
		cc.VisualColorGradingEnabled, cc.VisualBrightness, cc.VisualSaturation, cc.VisualContrast,
		cc.BrandingLogoEnabled, cc.BrandingLogoPath, cc.BrandingLogoPosition, cc.BrandingLogoOpacity, cc.BrandingLogoScale,
		cc.BrandingIntroEnabled, cc.BrandingIntroPath, cc.BrandingGenerateIntroFromLogo, cc.BrandingIntroDuration,
		cc.BrandingOutroEnabled, cc.BrandingOutroPath,
		cc.AudioMusicEnabled, cc.AudioMusicPath, cc.AudioMusicVolume, cc.AudioRandomMusic, cc.AudioMusicFolder,
		cc.PinnedCommentTemplate,
		cc.PreviewCaptionsEnabled, cc.PreviewCaptionStyle, cc.PreviewCaptionTextStyleJSON,
		cc.PreviewVideoOverlayConfigJSON, cc.PreviewTextOverlayConfigJSON,
		cc.MaxHighlights,
		now, cc.ChannelID,
	)
	if err != nil {
		return fmt.Errorf("update channel_config for channel %d: %w", cc.ChannelID, err)
	}
	return nil
}

// Kotlin ref: delete(channelId)
func (r *ChannelConfigRepo) Delete(ctx context.Context, channelID int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM channel_config WHERE channel_id = ?", channelID)
	if err != nil {
		return fmt.Errorf("delete channel_config for channel %d: %w", channelID, err)
	}
	return nil
}

func scanChannelConfig(row *sql.Row) (*ChannelConfig, error) {
	var cc ChannelConfig
	err := row.Scan(
		&cc.ID, &cc.ChannelID,
		&cc.GradientColorStart, &cc.GradientColorEnd,
		&cc.ThumbnailFontFamily, &cc.ThumbnailTextColor1, &cc.ThumbnailTextColor2, &cc.ThumbnailTextColor3,
		&cc.MadeForKids, &cc.DefaultCategoryID, &cc.DefaultPlaylistIDCortes, &cc.DefaultPlaylistIDShorts,
		&cc.DefaultTags, &cc.ShortsTitleHashtags, &cc.VideoDescriptionHashtags,
		&cc.AntiDuplicateEnabled, &cc.AntiDuplicateMode,
		&cc.SpeedEnabled, &cc.SpeedFactor,
		&cc.VisualCropEnabled, &cc.VisualCropPercent, &cc.VisualZoomEnabled, &cc.VisualZoomAmount,
		&cc.VisualColorGradingEnabled, &cc.VisualBrightness, &cc.VisualSaturation, &cc.VisualContrast,
		&cc.BrandingLogoEnabled, &cc.BrandingLogoPath, &cc.BrandingLogoPosition, &cc.BrandingLogoOpacity, &cc.BrandingLogoScale,
		&cc.BrandingIntroEnabled, &cc.BrandingIntroPath, &cc.BrandingGenerateIntroFromLogo, &cc.BrandingIntroDuration,
		&cc.BrandingOutroEnabled, &cc.BrandingOutroPath,
		&cc.AudioMusicEnabled, &cc.AudioMusicPath, &cc.AudioMusicVolume, &cc.AudioRandomMusic, &cc.AudioMusicFolder,
		&cc.PinnedCommentTemplate,
		&cc.PreviewCaptionsEnabled, &cc.PreviewCaptionStyle, &cc.PreviewCaptionTextStyleJSON,
		&cc.PreviewVideoOverlayConfigJSON, &cc.PreviewTextOverlayConfigJSON,
		&cc.MaxHighlights,
		&cc.CreatedAt, &cc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &cc, nil
}
