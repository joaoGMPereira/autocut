package thumbnail

// TemplateType identifies which thumbnail layout to use.
// Kotlin ref: ThumbnailStrategy.ThumbnailType
type TemplateType string

const (
	// TemplateBranded uses a static background image with a text overlay.
	// Kotlin ref: BrandedStrategy
	TemplateBranded TemplateType = "branded"

	// TemplateCentered places the source frame centred over a blurred background.
	// Kotlin ref: CenteredThumbnailStrategy
	TemplateCentered TemplateType = "centered"
)

// Anchor values for Position.Anchor.
const (
	AnchorTopLeft      = "top-left"
	AnchorTopCenter    = "top-center"
	AnchorTopRight     = "top-right"
	AnchorCenter       = "center"
	AnchorBottomLeft   = "bottom-left"
	AnchorBottomCenter = "bottom-center"
	AnchorBottomRight  = "bottom-right"
)

// Position describes where to place the text overlay on the canvas.
// Kotlin ref: TextPosition enum + TextPositionCalculator
type Position struct {
	// X is the horizontal pixel offset from the anchor point.
	X int
	// Y is the vertical pixel offset from the anchor point.
	Y int
	// Anchor is one of the AnchorXxx constants. Defaults to "center".
	Anchor string
}

// ffmpegX converts the anchor + offset to an FFmpeg drawtext x expression.
func (p Position) ffmpegX() string {
	switch p.Anchor {
	case AnchorTopLeft, AnchorBottomLeft:
		if p.X == 0 {
			return "0"
		}
		return itoa(p.X)
	case AnchorTopRight, AnchorBottomRight:
		return "w-text_w-" + itoa(abs(p.X))
	case AnchorTopCenter, AnchorBottomCenter, AnchorCenter, "":
		if p.X == 0 {
			return "(w-text_w)/2"
		}
		if p.X > 0 {
			return "(w-text_w)/2+" + itoa(p.X)
		}
		return "(w-text_w)/2-" + itoa(abs(p.X))
	default:
		return "(w-text_w)/2"
	}
}

// ffmpegY converts the anchor + offset to an FFmpeg drawtext y expression.
func (p Position) ffmpegY() string {
	switch p.Anchor {
	case AnchorTopLeft, AnchorTopCenter, AnchorTopRight:
		if p.Y == 0 {
			return "0"
		}
		return itoa(p.Y)
	case AnchorBottomLeft, AnchorBottomCenter, AnchorBottomRight:
		return "h-text_h-" + itoa(abs(p.Y))
	case AnchorCenter, "":
		if p.Y == 0 {
			return "(h-text_h)/2"
		}
		if p.Y > 0 {
			return "(h-text_h)/2+" + itoa(p.Y)
		}
		return "(h-text_h)/2-" + itoa(abs(p.Y))
	default:
		return "(h-text_h)/2"
	}
}

// BrandedConfig holds all parameters for a branded thumbnail.
// Kotlin ref: BrandedConfig + ThumbnailCreationConfig
type BrandedConfig struct {
	// BackgroundPath is the path to the static background image.
	BackgroundPath string
	// TextOverlay is the text to render on top of the background.
	TextOverlay string
	// FontPath is an optional absolute path to a .ttf/.otf font file.
	// When empty the ffmpeg drawtext filter uses its built-in font.
	FontPath string
	// FontColor is an FFmpeg colour string, e.g. "white" or "0xFFFFFF".
	FontColor string
	// FontSize in points. Default 80.
	FontSize int
	// Position controls where the text is drawn.
	Position Position
	// Width of the output image. Default 1280.
	Width int
	// Height of the output image. Default 720.
	Height int
}

// withDefaults returns a copy of c with zero-valued fields filled in.
func (c BrandedConfig) withDefaults() BrandedConfig {
	if c.FontColor == "" {
		c.FontColor = "white"
	}
	if c.FontSize == 0 {
		c.FontSize = 80
	}
	if c.Width == 0 {
		c.Width = 1280
	}
	if c.Height == 0 {
		c.Height = 720
	}
	if c.Position.Anchor == "" {
		c.Position.Anchor = AnchorCenter
	}
	return c
}

// CenteredConfig holds all parameters for a centered thumbnail.
// Kotlin ref: CenteredThumbnailStrategy + FFmpegCenteredThumbnailCreator
type CenteredConfig struct {
	// FramePath is the source image (usually an extracted video frame).
	FramePath string
	// TextOverlay is optional text drawn below the centred image.
	TextOverlay string
	// FontPath is an optional absolute path to a font file.
	FontPath string
	// FontColor is an FFmpeg colour string. Default "white".
	FontColor string
	// FontSize in points. Default 70.
	FontSize int
	// Width of the output image. Default 1280.
	Width int
	// Height of the output image. Default 720.
	Height int
}

// withDefaults returns a copy of c with zero-valued fields filled in.
func (c CenteredConfig) withDefaults() CenteredConfig {
	if c.FontColor == "" {
		c.FontColor = "white"
	}
	if c.FontSize == 0 {
		c.FontSize = 70
	}
	if c.Width == 0 {
		c.Width = 1280
	}
	if c.Height == 0 {
		c.Height = 720
	}
	return c
}

// --- small integer helpers (no import needed) ---

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
