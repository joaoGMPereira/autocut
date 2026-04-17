package thumbnail

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// StrategyType identifies the thumbnail generation strategy.
type StrategyType string

const (
	StrategyShorts    StrategyType = "shorts"
	StrategyLandscape StrategyType = "landscape"
)

// Strategy is the interface for pluggable thumbnail generation strategies.
type Strategy interface {
	Name() string
	Type() StrategyType
	Generate(ctx context.Context, input StrategyInput) (string, error)
}

// StrategyInput holds all data a Strategy needs to produce a single thumbnail.
type StrategyInput struct {
	SourcePath string           // video file (shorts) or base image (landscape)
	OutputPath string
	SeekSec    float64          // only used by shorts (frame extraction)
	FontPath   string
	Text       string
	Config     ThumbnailConfig
	LandConfig *LandscapeConfig // only used by landscape strategy
	ClipIndex  int              // 1-based position in batch
	TotalClips int
}

// LandscapeConfig holds visual settings specific to landscape (16:9) thumbnails.
type LandscapeConfig struct {
	GradientStart string           `json:"gradient_start"`
	GradientEnd   string           `json:"gradient_end"`
	BorderSize    int              `json:"border_size"`
	CornerRadius  int              `json:"corner_radius"`
	ApplyGradient bool             `json:"apply_gradient"`
	ApplyBlur     bool             `json:"apply_blur"`
	ApplyDarken   bool             `json:"apply_darken"`
	BaseImagePath string           `json:"base_image_path"`
	TextPosition  string           `json:"text_position"` // "top_right", "top_left", "center", "bottom_center"
	FontFamily    string           `json:"font_family"`
	FontSize      int              `json:"font_size"`
	TextColor     string           `json:"text_color"`
	OutlineColor  string           `json:"outline_color"`
	StrokeWidth   int              `json:"stroke_width"`
	ClipTexts      map[int64]string `json:"clip_texts"`
	ClipBaseImages map[int64]string `json:"clip_base_images"`
}

// DefaultLandscapeConfig returns a LandscapeConfig with sensible defaults.
func DefaultLandscapeConfig() LandscapeConfig {
	return LandscapeConfig{
		GradientStart: "#F27121",
		GradientEnd:   "#8A2387",
		BorderSize:    30,
		CornerRadius:  40,
		ApplyGradient: true,
		ApplyBlur:     false,
		ApplyDarken:   false,
		TextPosition:  "top_right",
		FontFamily:    "Impact",
		FontSize:      0, // auto
		TextColor:     "#FFD700",
		OutlineColor:  "#000000",
		StrokeWidth:   8,
	}
}

// Validate checks LandscapeConfig constraints. Returns nil if valid.
func (c *LandscapeConfig) Validate() error {
	if c.BorderSize < 0 || c.BorderSize > 60 {
		return fmt.Errorf("border_size must be 0–60")
	}
	if c.CornerRadius < 0 || c.CornerRadius > 80 {
		return fmt.Errorf("corner_radius must be 0–80")
	}
	if c.StrokeWidth < 0 || c.StrokeWidth > 20 {
		return fmt.Errorf("stroke_width must be 0–20")
	}
	if c.GradientStart != "" && !hexColorRe.MatchString(c.GradientStart) {
		return fmt.Errorf("gradient_start must be valid hex #RRGGBB")
	}
	if c.GradientEnd != "" && !hexColorRe.MatchString(c.GradientEnd) {
		return fmt.Errorf("gradient_end must be valid hex #RRGGBB")
	}
	if c.TextColor != "" && !hexColorRe.MatchString(c.TextColor) {
		return fmt.Errorf("text_color must be valid hex #RRGGBB")
	}
	if c.OutlineColor != "" && !hexColorRe.MatchString(c.OutlineColor) {
		return fmt.Errorf("outline_color must be valid hex #RRGGBB")
	}
	for _, t := range c.ClipTexts {
		if len(t) > 200 {
			return fmt.Errorf("clip text exceeds 200 characters")
		}
	}
	return nil
}

// TextForClip returns the per-clip text override from ClipTexts if set,
// otherwise auto-generates "PT{clipIndex}" uppercased.
func (c *LandscapeConfig) TextForClip(clipID int64, clipIndex int) string {
	if t, ok := c.ClipTexts[clipID]; ok {
		return strings.ToUpper(t)
	}
	return strings.ToUpper(fmt.Sprintf("PT%d", clipIndex))
}

// ThumbnailConfig holds all visual settings for thumbnail generation.
// Sent from the frontend per-request; not persisted as a DB entity.
type ThumbnailConfig struct {
	Text         string            `json:"text"`
	FontFamily   string            `json:"font_family"`
	FontSize     int               `json:"font_size"`
	TextColor    string            `json:"text_color"`
	OutlineColor string            `json:"outline_color"`
	StrokeWidth  int               `json:"stroke_width"`
	BlurEnabled  bool              `json:"blur_enabled"`
	BlurAmount   int               `json:"blur_amount"`
	DarkenEnabled bool             `json:"darken_enabled"`
	DarkenAmount float64           `json:"darken_amount"`
	CenterScale  float64           `json:"center_scale"`
	CornerRadius int               `json:"corner_radius"`
	TextYOffset  int               `json:"text_y_offset"`
	ClipTexts    map[int64]string  `json:"clip_texts"`
}

// DefaultConfig returns a ThumbnailConfig with production defaults matching the Kotlin reference.
func DefaultConfig() ThumbnailConfig {
	return ThumbnailConfig{
		Text:          "",
		FontFamily:    "Impact",
		FontSize:      0, // auto
		TextColor:     "#FFFFFF",
		OutlineColor:  "#000000",
		StrokeWidth:   4,
		BlurEnabled:   true,
		BlurAmount:    25,
		DarkenEnabled: true,
		DarkenAmount:  0.30,
		CenterScale:   0.75,
		CornerRadius:  40,
		TextYOffset:   321,
		ClipTexts:     nil,
	}
}

var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Validate checks config constraints. Returns nil if valid.
func (c *ThumbnailConfig) Validate() error {
	if len(c.Text) > 200 {
		return fmt.Errorf("text exceeds 200 characters")
	}
	if c.FontSize != 0 && (c.FontSize < 20 || c.FontSize > 200) {
		return fmt.Errorf("font_size must be 0 (auto) or 20–200")
	}
	if c.StrokeWidth < 0 || c.StrokeWidth > 20 {
		return fmt.Errorf("stroke_width must be 0–20")
	}
	if c.BlurAmount < 1 || c.BlurAmount > 100 {
		return fmt.Errorf("blur_amount must be 1–100")
	}
	if c.DarkenAmount < 0 || c.DarkenAmount > 1.0 {
		return fmt.Errorf("darken_amount must be 0.0–1.0")
	}
	if c.CenterScale < 0.3 || c.CenterScale > 1.0 {
		return fmt.Errorf("center_scale must be 0.3–1.0")
	}
	if c.CornerRadius < 0 || c.CornerRadius > 100 {
		return fmt.Errorf("corner_radius must be 0–100")
	}
	if c.TextYOffset < -500 || c.TextYOffset > 500 {
		return fmt.Errorf("text_y_offset must be -500–500")
	}
	if c.TextColor != "" && !hexColorRe.MatchString(c.TextColor) {
		return fmt.Errorf("text_color must be valid hex #RRGGBB")
	}
	if c.OutlineColor != "" && !hexColorRe.MatchString(c.OutlineColor) {
		return fmt.Errorf("outline_color must be valid hex #RRGGBB")
	}
	return nil
}

// TextForClip returns the per-clip text override or the global text, uppercased.
func (c *ThumbnailConfig) TextForClip(clipID int64) string {
	if t, ok := c.ClipTexts[clipID]; ok {
		return strings.ToUpper(t)
	}
	return strings.ToUpper(c.Text)
}

// ThumbnailTemplate is a named preset stored in app_settings.
type ThumbnailTemplate struct {
	Name         string           `json:"name"`
	Config       ThumbnailConfig  `json:"config"`
	LandConfig   *LandscapeConfig `json:"landscape_config,omitempty"`
	StrategyType StrategyType     `json:"strategy_type"`
	CreatedAt    int64            `json:"created_at"`
}

// GenerateRequest is the HTTP request body for batch or single-clip generation.
type GenerateRequest struct {
	Config          ThumbnailConfig  `json:"config"`
	LandscapeConfig *LandscapeConfig `json:"landscape_config,omitempty"`
	ThumbnailURL    string           `json:"thumbnail_url,omitempty"`
}

// BatchGenerateResponse is returned for a batch generation request (202).
type BatchGenerateResponse struct {
	JobID      string `json:"job_id"`
	TotalClips int    `json:"total_clips"`
}

// SingleGenerateResponse is returned for a single-clip generation (200).
type SingleGenerateResponse struct {
	ClipID        int64  `json:"clip_id"`
	ThumbnailPath string `json:"thumbnail_path"`
}

// FontInfo describes a detected system font.
type FontInfo struct {
	Family string `json:"family"`
	Path   string `json:"path"`
}

// FontListResponse is the HTTP response for GET /api/thumbnail/fonts.
type FontListResponse struct {
	Fonts       []FontInfo `json:"fonts"`
	DefaultFont string     `json:"default_font"`
}
