package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CaptionTextStyle holds parsed caption styling from PreviewCaptionTextStyleJSON.
type CaptionTextStyle struct {
	FontFamily   string `json:"font_family"`
	FontSize     int    `json:"font_size"`
	PrimaryColor string `json:"primary_color"` // hex e.g. "FFFFFF"
	OutlineColor string `json:"outline_color"`
	OutlineWidth int    `json:"outline_width"`
	BgEnabled    bool   `json:"bg_enabled"`
	BgColor      string `json:"bg_color"`
	Shadow       int    `json:"shadow"`
	Alignment    int    `json:"alignment"` // ASS alignment (2 = bottom-center)
	MarginV      int    `json:"margin_v"`
}

// GenerateASS creates a minimal ASS subtitle file for preview purposes.
// Returns the file path and a cleanup function that deletes the temp file.
func GenerateASS(style CaptionTextStyle) (string, func(), error) {
	// Defaults
	if style.FontFamily == "" {
		style.FontFamily = "Arial"
	}
	if style.FontSize <= 0 {
		style.FontSize = 48
	}
	if style.PrimaryColor == "" {
		style.PrimaryColor = "FFFFFF"
	}
	if style.OutlineColor == "" {
		style.OutlineColor = "000000"
	}
	if style.Alignment <= 0 {
		style.Alignment = 2 // bottom-center
	}

	// Convert hex colors to ASS format (&HBBGGRR)
	primaryASS := hexToASS(style.PrimaryColor)
	outlineASS := hexToASS(style.OutlineColor)

	// Background color (outline box)
	bgStyle := "0" // no background box by default
	bgColorASS := "&H00000000"
	if style.BgEnabled {
		bgStyle = "3" // opaque box
		if style.BgColor != "" {
			bgColorASS = "&H80" + hexToASSBGR(style.BgColor)
		}
	}

	shadow := style.Shadow
	if shadow < 0 {
		shadow = 0
	}

	ass := fmt.Sprintf(`[Script Info]
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,%s,%d,%s,&H000000FF,%s,%s,0,0,0,0,100,100,0,0,%s,%d,%d,%d,10,10,%d,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
Dialogue: 0,0:00:00.00,0:00:05.00,Default,,0,0,0,,Sample caption text
Dialogue: 0,0:00:05.00,0:00:10.00,Default,,0,0,0,,Preview caption style
Dialogue: 0,0:00:10.00,0:00:15.00,Default,,0,0,0,,This is how captions look
Dialogue: 0,0:00:15.00,0:00:20.00,Default,,0,0,0,,With your chosen settings
Dialogue: 0,0:00:20.00,0:00:25.00,Default,,0,0,0,,Sample caption text
Dialogue: 0,0:00:25.00,0:00:30.00,Default,,0,0,0,,Preview caption style
Dialogue: 0,0:00:30.00,0:00:35.00,Default,,0,0,0,,This is how captions look
Dialogue: 0,0:00:35.00,0:00:40.00,Default,,0,0,0,,With your chosen settings
Dialogue: 0,0:00:40.00,0:00:45.00,Default,,0,0,0,,Sample caption text
Dialogue: 0,0:00:45.00,0:00:50.00,Default,,0,0,0,,Preview caption style
Dialogue: 0,0:00:50.00,0:00:55.00,Default,,0,0,0,,This is how captions look
Dialogue: 0,0:00:55.00,0:01:00.00,Default,,0,0,0,,With your chosen settings
Dialogue: 0,0:01:00.00,0:01:05.00,Default,,0,0,0,,Sample caption text
Dialogue: 0,0:01:05.00,0:01:10.00,Default,,0,0,0,,Preview caption style
Dialogue: 0,0:01:10.00,0:01:15.00,Default,,0,0,0,,This is how captions look
Dialogue: 0,0:01:15.00,0:01:20.00,Default,,0,0,0,,With your chosen settings
Dialogue: 0,0:01:20.00,0:01:25.00,Default,,0,0,0,,Sample caption text
Dialogue: 0,0:01:25.00,0:01:30.00,Default,,0,0,0,,Preview caption style
Dialogue: 0,0:01:30.00,0:01:35.00,Default,,0,0,0,,This is how captions look
Dialogue: 0,0:01:35.00,0:01:40.00,Default,,0,0,0,,With your chosen settings
Dialogue: 0,0:01:40.00,0:01:45.00,Default,,0,0,0,,Sample caption text
Dialogue: 0,0:01:45.00,0:01:50.00,Default,,0,0,0,,Preview caption style
Dialogue: 0,0:01:50.00,0:01:55.00,Default,,0,0,0,,This is how captions look
Dialogue: 0,0:01:55.00,0:02:00.00,Default,,0,0,0,,With your chosen settings
`,
		style.FontFamily, style.FontSize,
		primaryASS, outlineASS, bgColorASS,
		bgStyle, style.OutlineWidth, shadow, style.Alignment,
		style.MarginV,
	)

	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("preview_captions_%d.ass", os.Getpid()))

	if err := os.WriteFile(tmpFile, []byte(ass), 0o644); err != nil {
		return "", nil, fmt.Errorf("write ASS file: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(tmpFile)
	}

	return tmpFile, cleanup, nil
}

// hexToASS converts a hex color (e.g. "FFFFFF") to ASS format "&H00BBGGRR".
func hexToASS(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) < 6 {
		hex = "FFFFFF"
	}
	r := hex[0:2]
	g := hex[2:4]
	b := hex[4:6]
	return fmt.Sprintf("&H00%s%s%s", b, g, r)
}

// hexToASSBGR converts a hex color (e.g. "FFFFFF") to just the BGR portion.
func hexToASSBGR(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) < 6 {
		hex = "000000"
	}
	r := hex[0:2]
	g := hex[2:4]
	b := hex[4:6]
	return fmt.Sprintf("%s%s%s", b, g, r)
}
