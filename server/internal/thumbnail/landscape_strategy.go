package thumbnail

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// LandscapeStrategy generates 16:9 thumbnails with gradient border + text overlay.
// Uses ImageMagick for rendering (ported from Kotlin BrandedStrategy).
type LandscapeStrategy struct{}

func (s *LandscapeStrategy) Name() string       { return "landscape" }
func (s *LandscapeStrategy) Type() StrategyType { return StrategyLandscape }

func (s *LandscapeStrategy) Generate(ctx context.Context, input StrategyInput) (string, error) {
	if input.SourcePath == "" {
		return "", fmt.Errorf("source path is required")
	}
	if _, err := os.Stat(input.SourcePath); err != nil {
		return "", fmt.Errorf("source not found: %s", input.SourcePath)
	}

	lc := input.LandConfig
	if lc == nil {
		def := DefaultLandscapeConfig()
		lc = &def
	}

	// Resolve text: LandConfig overrides take priority, then input.Text, then auto PT{N}.
	text := input.Text
	if text == "" {
		text = strings.ToUpper(fmt.Sprintf("PT%d", input.ClipIndex))
	}

	args := buildLandscapeArgs(input.SourcePath, input.OutputPath, text, lc, input.FontPath)

	if err := RunImageMagick(ctx, args); err != nil {
		return "", fmt.Errorf("imagemagick thumbnail: %w", err)
	}

	return input.OutputPath, nil
}

// buildLandscapeArgs constructs the full ImageMagick command for landscape thumbnails.
// Supports two modes: gradient border (branded) and simple resize (no gradient).
func buildLandscapeArgs(sourcePath, outputPath, text string, lc *LandscapeConfig, fontPath string) []string {
	binary := ImageMagickBinary()
	args := []string{binary}

	effectiveBorder := lc.BorderSize
	if !lc.ApplyGradient {
		effectiveBorder = 0
	}
	innerW := LandscapeWidth - (effectiveBorder * 2)
	innerH := LandscapeHeight - (effectiveBorder * 2)

	if !lc.ApplyGradient {
		// No gradient: simple resize + effects + text
		args = append(args, sourcePath,
			"-filter", "Lanczos",
			"-resize", fmt.Sprintf("%dx%d^", LandscapeWidth, LandscapeHeight),
			"-gravity", "center",
			"-extent", fmt.Sprintf("%dx%d", LandscapeWidth, LandscapeHeight),
			"-unsharp", "0.5x0.5+0.5+0.008",
		)
		if lc.ApplyBlur {
			args = append(args, "-blur", "0x3")
		}
		if lc.ApplyDarken {
			args = append(args, "-fill", "#120520", "-colorize", "60%")
		}
	} else {
		// Gradient mode
		// 1. Gradient background
		args = append(args,
			"(", "-size", fmt.Sprintf("%dx%d", LandscapeWidth, LandscapeHeight),
			fmt.Sprintf("gradient:%s-%s", lc.GradientStart, lc.GradientEnd), ")",
		)

		// 2. Inner content with rounded corners
		innerArgs := []string{
			"(", sourcePath,
			"-filter", "Lanczos",
			"-resize", fmt.Sprintf("%dx%d^", innerW, innerH),
			"-gravity", "center",
			"-extent", fmt.Sprintf("%dx%d", innerW, innerH),
			"-unsharp", "0.5x0.5+0.5+0.008",
		}
		if lc.ApplyBlur {
			innerArgs = append(innerArgs, "-blur", "0x3")
		}
		if lc.ApplyDarken {
			innerArgs = append(innerArgs, "-fill", "#120520", "-colorize", "60%")
		}

		// Rounded corner mask
		innerArgs = append(innerArgs,
			"-alpha", "set",
			"(", "+clone",
			"-alpha", "transparent",
			"-fill", "white",
			"-draw", fmt.Sprintf("roundrectangle 0,0 %d,%d %d,%d", innerW, innerH, lc.CornerRadius, lc.CornerRadius),
			")",
			"-alpha", "off", "-compose", "CopyOpacity", "-composite",
			")",
		)
		args = append(args, innerArgs...)

		// 3. Composite
		args = append(args, "-compose", "Over", "-gravity", "center", "-composite")
	}

	// 4. Text overlay (two-pass: stroke then fill)
	if text != "" {
		fontSize := lc.FontSize
		if fontSize == 0 {
			fontSize = autoLandscapeFontSize(text)
		}
		gravity := textPositionToGravity(lc.TextPosition)
		offset := textPositionToOffset(lc.TextPosition)

		args = append(args,
			"-gravity", gravity,
			"-font", fontPath,
			"-pointsize", fmt.Sprintf("%d", fontSize),
			// Stroke pass
			"-fill", lc.OutlineColor,
			"-stroke", lc.OutlineColor,
			"-strokewidth", fmt.Sprintf("%d", lc.StrokeWidth),
			"-annotate", offset, text,
			// Fill pass
			"-fill", lc.TextColor,
			"-stroke", "none",
			"-annotate", offset, text,
		)
	}

	args = append(args, "-sampling-factor", "1x1", "-quality", "98", outputPath)
	return args
}

// textPositionToGravity maps a TextPosition string to ImageMagick gravity.
func textPositionToGravity(pos string) string {
	switch pos {
	case "top_right":
		return "NorthEast"
	case "top_left":
		return "NorthWest"
	case "center":
		return "Center"
	case "bottom_center":
		return "South"
	default:
		return "NorthEast"
	}
}

// textPositionToOffset maps a TextPosition string to an ImageMagick annotate offset.
func textPositionToOffset(pos string) string {
	switch pos {
	case "top_right":
		return "+50+50"
	case "top_left":
		return "+50+50"
	case "center":
		return "+0+0"
	case "bottom_center":
		return "+0+50"
	default:
		return "+50+50"
	}
}

// autoLandscapeFontSize calculates font size based on text length for landscape thumbnails.
func autoLandscapeFontSize(text string) int {
	n := len(text)
	switch {
	case n <= 5:
		return 120
	case n <= 15:
		return 100
	case n <= 30:
		return 80
	default:
		return 60
	}
}
