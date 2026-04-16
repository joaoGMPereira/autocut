package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CaptionTextStyle holds parsed caption styling from PreviewCaptionTextStyleJSON.
type CaptionTextStyle struct {
	FontFamily   string `json:"font_family"`
	FontSize     int    `json:"font_size"`
	Bold         bool   `json:"bold"`
	Italic       bool   `json:"italic"`
	PrimaryColor string `json:"primary_color"` // hex e.g. "FFFFFF"
	OutlineColor string `json:"outline_color"`
	OutlineWidth int    `json:"outline_width"`
	BgEnabled    bool   `json:"bg_enabled"`
	BgColor      string `json:"bg_color"`
	Shadow       int    `json:"shadow"`
	Alignment    int    `json:"alignment"` // ASS alignment (2 = bottom-center)
	MarginV      int    `json:"margin_v"`
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
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
Style: Default,%s,%d,%s,&H000000FF,%s,%s,%d,%d,0,0,100,100,0,0,%s,%d,%d,%d,10,10,%d,1

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
		boolToInt(style.Bold), boolToInt(style.Italic),
		bgStyle, style.OutlineWidth, shadow, style.Alignment,
		style.MarginV,
	)

	f, err := os.CreateTemp("", "captions_*.ass")
	if err != nil {
		return "", nil, fmt.Errorf("create ASS temp file: %w", err)
	}
	tmpFile := f.Name()
	if _, err := f.WriteString(ass); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpFile)
		return "", nil, fmt.Errorf("write ASS file: %w", err)
	}
	_ = f.Close()

	cleanup := func() {
		_ = os.Remove(tmpFile)
	}

	return tmpFile, cleanup, nil
}

// secToASS converts seconds to ASS time format H:MM:SS.CS (centiseconds).
func secToASS(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

// transcriptSegmentV1 is the flat-array format: [{start, end, text}] with times in seconds.
type transcriptSegmentV1 struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// transcriptSegmentV2 is the segment inside the object format, times in nanoseconds.
type transcriptSegmentV2 struct {
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

// transcriptV2 is the outer object format: {segments, language, duration}.
type transcriptV2 struct {
	Segments []transcriptSegmentV2 `json:"segments"`
	Language string                `json:"language"`
	Duration float64               `json:"duration"`
}

// transcriptWhisperRaw is the raw whisper-cli -oj output format.
type transcriptWhisperRaw struct {
	Transcription []struct {
		Offsets struct {
			From int64 `json:"from"` // milliseconds
			To   int64 `json:"to"`
		} `json:"offsets"`
		Text string `json:"text"`
	} `json:"transcription"`
}

// GenerateASSFromTranscript creates an ASS subtitle file using real transcript data.
// It reads the transcript at transcriptPath, filters to the [startSec, startSec+durationSec]
// window, shifts times to be relative to startSec, and emits real dialogue lines.
// Falls back to GenerateASS if the transcript is missing, unparseable, or has no segments
// in the preview window.
func GenerateASSFromTranscript(transcriptPath string, startSec int, durationSec int64, style CaptionTextStyle) (string, func(), error) {
	// Fall back immediately if no path given
	if transcriptPath == "" {
		return GenerateASS(style)
	}

	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		// File not found or unreadable — fall back gracefully
		return GenerateASS(style)
	}

	// Parse transcript: try format 1 (array) first, then format 2 (object with segments)
	var segments []transcriptSegmentV1

	if err := json.Unmarshal(data, &segments); err != nil || len(segments) == 0 {
		// Not a flat array — try object format (v2: {segments})
		var v2 transcriptV2
		if err2 := json.Unmarshal(data, &v2); err2 == nil && len(v2.Segments) > 0 {
			segments = make([]transcriptSegmentV1, 0, len(v2.Segments))
			for _, s := range v2.Segments {
				segments = append(segments, transcriptSegmentV1{
					Start: s.Start / 1e9,
					End:   s.End / 1e9,
					Text:  s.Text,
				})
			}
		} else {
			// Try whisper raw format (v3: {transcription: [{offsets: {from, to}, text}]})
			var raw transcriptWhisperRaw
			if err3 := json.Unmarshal(data, &raw); err3 != nil || len(raw.Transcription) == 0 {
				return GenerateASS(style)
			}
			segments = make([]transcriptSegmentV1, 0, len(raw.Transcription))
			for _, t := range raw.Transcription {
				text := strings.TrimSpace(t.Text)
				if text == "" {
					continue
				}
				segments = append(segments, transcriptSegmentV1{
					Start: float64(t.Offsets.From) / 1000.0,
					End:   float64(t.Offsets.To) / 1000.0,
					Text:  text,
				})
			}
		}
	}

	// Filter segments to the preview window
	windowStart := float64(startSec)
	windowEnd := float64(startSec) + float64(durationSec)

	var filtered []transcriptSegmentV1
	for _, seg := range segments {
		if seg.End >= windowStart && seg.Start <= windowEnd {
			filtered = append(filtered, seg)
		}
	}

	if len(filtered) == 0 {
		return GenerateASS(style)
	}

	// Apply style defaults
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
		style.Alignment = 2
	}

	primaryASS := hexToASS(style.PrimaryColor)
	outlineASS := hexToASS(style.OutlineColor)

	bgStyle := "0"
	bgColorASS := "&H00000000"
	if style.BgEnabled {
		bgStyle = "3"
		if style.BgColor != "" {
			bgColorASS = "&H80" + hexToASSBGR(style.BgColor)
		}
	}

	shadow := style.Shadow
	if shadow < 0 {
		shadow = 0
	}

	header := fmt.Sprintf(`[Script Info]
ScriptType: v4.00+
PlayResX: 1920
PlayResY: 1080

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,%s,%d,%s,&H000000FF,%s,%s,%d,%d,0,0,100,100,0,0,%s,%d,%d,%d,10,10,%d,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`,
		style.FontFamily, style.FontSize,
		primaryASS, outlineASS, bgColorASS,
		boolToInt(style.Bold), boolToInt(style.Italic),
		bgStyle, style.OutlineWidth, shadow, style.Alignment,
		style.MarginV,
	)

	var dialogues strings.Builder
	dialogues.WriteString(header)

	for _, seg := range filtered {
		adjStart := seg.Start - windowStart
		adjEnd := seg.End - windowStart

		// Clamp to [0, durationSec]
		if adjStart < 0 {
			adjStart = 0
		}
		if adjEnd > float64(durationSec) {
			adjEnd = float64(durationSec)
		}
		if adjEnd <= adjStart {
			continue
		}

		text := strings.ReplaceAll(seg.Text, "\n", " ")
		dialogues.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n",
			secToASS(adjStart), secToASS(adjEnd), text))
	}

	f, err := os.CreateTemp("", "captions_*.ass")
	if err != nil {
		return "", nil, fmt.Errorf("create ASS temp file: %w", err)
	}
	tmpFile := f.Name()
	if _, err := f.WriteString(dialogues.String()); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpFile)
		return "", nil, fmt.Errorf("write ASS file: %w", err)
	}
	_ = f.Close()

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

// FindWhisperModel returns the path to the first ggml-*.bin model file in
// {dataDir}/models/whisper, or "" if none found.
func FindWhisperModel(dataDir string) string {
	dir := filepath.Join(dataDir, "models", "whisper")
	matches, err := filepath.Glob(filepath.Join(dir, "ggml-*.bin"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// whisperSegment is the JSON shape emitted by whisper-cli -oj.
type whisperSegment struct {
	Offsets struct {
		From int64 `json:"from"` // milliseconds from segment start
		To   int64 `json:"to"`
	} `json:"offsets"`
	Text string `json:"text"`
}

type whisperOutput struct {
	Transcription []whisperSegment `json:"transcription"`
}

// ResolveTranscript returns a usable transcript path for caption burn-in.
// If captionsEnabled is false, returns ("", nil, nil).
// If transcriptPath is already set and the file exists, returns it as-is (no cleanup).
// If no transcript exists but a Whisper model is available, transcribes the segment on-demand.
// Otherwise returns ("", nil, nil) — captions will fall back to placeholder via GenerateASS.
func ResolveTranscript(videoPath string, startSec int, durationSec int64, dataDir, transcriptPath string, captionsEnabled bool) (string, func(), error) {
	if !captionsEnabled {
		return "", nil, nil
	}
	if transcriptPath != "" {
		if _, err := os.Stat(transcriptPath); err == nil {
			return transcriptPath, nil, nil
		}
	}
	modelPath := FindWhisperModel(dataDir)
	if modelPath == "" {
		return "", nil, nil
	}
	return TranscribeSegment(videoPath, startSec, durationSec, modelPath)
}

// TranscribeSegment extracts audio from videoPath at [startSec, startSec+durationSec],
// runs whisper-cli on it, and returns a path to a temporary V1 transcript JSON file.
// Timestamps in the output are absolute (relative to the full video, not the segment).
// The caller must call the returned cleanup func when done.
func TranscribeSegment(videoPath string, startSec int, durationSec int64, modelPath string) (transcriptPath string, cleanup func(), err error) {
	pid := os.Getpid()
	audioPath := filepath.Join(os.TempDir(), fmt.Sprintf("preview_audio_%d_%d.wav", pid, startSec))
	whisperOutBase := filepath.Join(os.TempDir(), fmt.Sprintf("preview_whisper_%d_%d", pid, startSec))
	transcriptOut := filepath.Join(os.TempDir(), fmt.Sprintf("preview_transcript_%d_%d.json", pid, startSec))

	cleanupFn := func() {
		_ = os.Remove(audioPath)
		_ = os.Remove(whisperOutBase + ".json")
		_ = os.Remove(transcriptOut)
	}

	// Extract 16kHz mono WAV — whisper.cpp requires wav input
	audioArgs := []string{
		"-y",
		"-ss", fmt.Sprintf("%d", startSec),
		"-i", videoPath,
		"-t", fmt.Sprintf("%d", durationSec),
		"-vn", "-ar", "16000", "-ac", "1", "-f", "wav",
		audioPath,
	}
	if out, cmdErr := exec.Command("ffmpeg", audioArgs...).CombinedOutput(); cmdErr != nil {
		return "", cleanupFn, fmt.Errorf("extract audio for whisper: %w: %s", cmdErr, out)
	}

	// Run whisper-cli with tuned params (ported from Kotlin project).
	whisperArgs := []string{
		"-m", modelPath,
		"-oj", "-of", whisperOutBase,
		"-l", "auto",
		"--beam-size", "8",
		"--best-of", "5",
		"--entropy-thold", "2.4",
		"--max-context", "0",
		"--max-len", "80",
		"--split-on-word",
		"--suppress-nst",
		audioPath,
	}
	if out, cmdErr := exec.Command("whisper-cli", whisperArgs...).CombinedOutput(); cmdErr != nil {
		return "", cleanupFn, fmt.Errorf("whisper-cli: %w: %s", cmdErr, out)
	}

	// Parse whisper JSON
	data, err := os.ReadFile(whisperOutBase + ".json")
	if err != nil {
		return "", cleanupFn, fmt.Errorf("read whisper output: %w", err)
	}
	var result whisperOutput
	if err := json.Unmarshal(data, &result); err != nil {
		return "", cleanupFn, fmt.Errorf("parse whisper output: %w", err)
	}

	// Convert to V1 format; shift offsets to be absolute (add startSec)
	segments := make([]transcriptSegmentV1, 0, len(result.Transcription))
	offset := float64(startSec)
	for _, t := range result.Transcription {
		text := strings.TrimSpace(t.Text)
		if text == "" {
			continue
		}
		segments = append(segments, transcriptSegmentV1{
			Start: float64(t.Offsets.From)/1000.0 + offset,
			End:   float64(t.Offsets.To)/1000.0 + offset,
			Text:  text,
		})
	}

	out, err := json.Marshal(segments)
	if err != nil {
		return "", cleanupFn, fmt.Errorf("marshal transcript: %w", err)
	}
	if err := os.WriteFile(transcriptOut, out, 0o644); err != nil {
		return "", cleanupFn, fmt.Errorf("write transcript: %w", err)
	}

	return transcriptOut, cleanupFn, nil
}
