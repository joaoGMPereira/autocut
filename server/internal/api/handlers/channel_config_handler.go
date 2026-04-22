package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ChannelConfigHandler serves GET /api/channels/{id}/config
type ChannelConfigHandler struct {
	repo *database.ChannelConfigRepo
	log  *slog.Logger
}

// NewChannelConfigHandler constructs a ChannelConfigHandler.
func NewChannelConfigHandler(repo *database.ChannelConfigRepo) *ChannelConfigHandler {
	return &ChannelConfigHandler{repo: repo, log: slog.With("handler", "channel_config")}
}

// channelConfigJSON is the serialisable shape of a ChannelConfig.
// sql.NullString → *string so JSON omits null fields cleanly.
type channelConfigJSON struct {
	ID                           int64   `json:"id"`
	ChannelID                    int64   `json:"channel_id"`
	MadeForKids                  bool    `json:"made_for_kids"`
	DefaultCategoryID            int     `json:"default_category_id"`
	DefaultPlaylistIDCortes      *string `json:"default_playlist_id_cortes"`
	DefaultPlaylistIDShorts      *string `json:"default_playlist_id_shorts"`
	DefaultTags                  string  `json:"default_tags"`
	ShortsTitleHashtags          string  `json:"shorts_title_hashtags"`
	VideoDescriptionHashtags     string  `json:"video_description_hashtags"`
	GradientColorStart           string  `json:"gradient_color_start"`
	GradientColorEnd             string  `json:"gradient_color_end"`
	ThumbnailFontFamily          string  `json:"thumbnail_font_family"`
	SpeedEnabled                 bool    `json:"speed_enabled"`
	SpeedFactor                  float64 `json:"speed_factor"`
	AntiDuplicateEnabled         bool    `json:"anti_duplicate_enabled"`
	AntiDuplicateMode            string  `json:"anti_duplicate_mode"`
	VisualCropEnabled            bool    `json:"visual_crop_enabled"`
	VisualZoomEnabled            bool    `json:"visual_zoom_enabled"`
	VisualColorGradingEnabled    bool    `json:"visual_color_grading_enabled"`
	BrandingLogoEnabled          bool    `json:"branding_logo_enabled"`
	BrandingLogoPath             *string `json:"branding_logo_path"`
	BrandingIntroEnabled         bool    `json:"branding_intro_enabled"`
	BrandingOutroEnabled         bool    `json:"branding_outro_enabled"`
	AudioMusicEnabled            bool    `json:"audio_music_enabled"`
	AudioMusicVolume             float64 `json:"audio_music_volume"`
	PreviewCaptionsEnabled       bool    `json:"preview_captions_enabled"`
	PreviewCaptionStyle          string  `json:"preview_caption_style"`
	MaxHighlights                *int64  `json:"max_highlights"`
	PinnedCommentTemplate        string  `json:"pinned_comment_template"`
}

func toChannelConfigJSON(cc *database.ChannelConfig) channelConfigJSON {
	j := channelConfigJSON{
		ID:                        cc.ID,
		ChannelID:                 cc.ChannelID,
		MadeForKids:               cc.MadeForKids,
		DefaultCategoryID:         cc.DefaultCategoryID,
		DefaultTags:               cc.DefaultTags,
		ShortsTitleHashtags:       cc.ShortsTitleHashtags,
		VideoDescriptionHashtags:  cc.VideoDescriptionHashtags,
		GradientColorStart:        cc.GradientColorStart,
		GradientColorEnd:          cc.GradientColorEnd,
		ThumbnailFontFamily:       cc.ThumbnailFontFamily,
		SpeedEnabled:              cc.SpeedEnabled,
		SpeedFactor:               cc.SpeedFactor,
		AntiDuplicateEnabled:      cc.AntiDuplicateEnabled,
		AntiDuplicateMode:         cc.AntiDuplicateMode,
		VisualCropEnabled:         cc.VisualCropEnabled,
		VisualZoomEnabled:         cc.VisualZoomEnabled,
		VisualColorGradingEnabled: cc.VisualColorGradingEnabled,
		BrandingLogoEnabled:       cc.BrandingLogoEnabled,
		BrandingIntroEnabled:      cc.BrandingIntroEnabled,
		BrandingOutroEnabled:      cc.BrandingOutroEnabled,
		AudioMusicEnabled:         cc.AudioMusicEnabled,
		AudioMusicVolume:          cc.AudioMusicVolume,
		PreviewCaptionsEnabled:    cc.PreviewCaptionsEnabled,
		PreviewCaptionStyle:       cc.PreviewCaptionStyle,
		PinnedCommentTemplate:     cc.PinnedCommentTemplate,
	}
	if cc.DefaultPlaylistIDCortes.Valid {
		j.DefaultPlaylistIDCortes = &cc.DefaultPlaylistIDCortes.String
	}
	if cc.DefaultPlaylistIDShorts.Valid {
		j.DefaultPlaylistIDShorts = &cc.DefaultPlaylistIDShorts.String
	}
	if cc.BrandingLogoPath.Valid {
		j.BrandingLogoPath = &cc.BrandingLogoPath.String
	}
	if cc.MaxHighlights.Valid {
		j.MaxHighlights = &cc.MaxHighlights.Int64
	}
	return j
}

// GetConfig handles GET /api/channels/{id}/config
func (h *ChannelConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	cc, err := h.repo.GetByChannelID(r.Context(), id)
	if err != nil {
		h.log.Error("get channel config", "channel_id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if cc == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toChannelConfigJSON(cc))
}
