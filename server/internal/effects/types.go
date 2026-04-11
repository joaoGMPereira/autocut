// Package effects provides video post-processing effects via FFmpeg.
// Each effect is a stateless operation: input path(s) → FFmpeg filter → output path.
package effects

// TextPosition defines named anchor points for text/logo overlays.
type TextPosition string

const (
	PosTopLeft      TextPosition = "top_left"
	PosTopCenter    TextPosition = "top_center"
	PosTopRight     TextPosition = "top_right"
	PosMidLeft      TextPosition = "mid_left"
	PosMidCenter    TextPosition = "mid_center"
	PosBottomLeft   TextPosition = "bottom_left"
	PosBottomCenter TextPosition = "bottom_center"
	PosBottomRight  TextPosition = "bottom_right"
)

// SpeedSegment defines a time range and the playback speed to apply within it.
// Speed 0.5 = half speed (slow motion), 2.0 = double speed.
// Speed must be > 0.
type SpeedSegment struct {
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
	Speed    float64 `json:"speed"`
}

// TransitionType selects the visual transition style between clips.
type TransitionType string

const (
	// TransitionXfade uses FFmpeg's xfade filter (crossfade blend).
	TransitionXfade TransitionType = "xfade"
	// TransitionDipBlack fades to black then fades in the next clip.
	TransitionDipBlack TransitionType = "dip_black"
	// TransitionFade is an alias for xfade with the "fade" transition style.
	TransitionFade TransitionType = "fade"
)
