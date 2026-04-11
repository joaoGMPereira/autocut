package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"unicode"
)

// ChatHighlight is a time window with high chat activity detected from Twitch
// chat data exported by TwitchDownloaderCLI.
type ChatHighlight struct {
	StartSec float64 `json:"start_sec"`
	EndSec   float64 `json:"end_sec"`
	Score    float64 `json:"score"` // normalised density, 0.0–1.0
	Reason   string  `json:"reason"` // "emote_spike" | "caps_excitement"
}

// twitchChatMessage is the subset of TwitchDownloaderCLI JSON fields that the
// detector uses. The full schema has more fields; unused ones are ignored.
type twitchChatMessage struct {
	ContentOffsetSeconds float64 `json:"content_offset_seconds"`
	Message              struct {
		Body string `json:"body"`
	} `json:"message"`
}

// ChatActivityDetector analyses a TwitchDownloaderCLI JSON chat export to find
// time windows with unusually high audience activity (emote spam, caps lock).
type ChatActivityDetector struct {
	logger *slog.Logger
}

// NewChatActivityDetector creates a ChatActivityDetector.
func NewChatActivityDetector() *ChatActivityDetector {
	return &ChatActivityDetector{
		logger: slog.With("component", "chat_activity_detector"),
	}
}

const (
	chatWindowSec    = 10.0 // sliding window width in seconds
	chatWindowStep   = 5.0  // step between window starts in seconds
	chatDensityThreshold = 0.3 // normalised density above which a window is a highlight
	chatMergeGapSec  = 5.0  // adjacent windows within this gap are merged
)

// DetectHighlights reads the JSON chat file at chatJSONPath and returns
// ChatHighlight entries for windows where emote/caps density exceeds the threshold.
//
// TwitchDownloaderCLI JSON format:
//
//	[{"content_offset_seconds": 12.3, "message": {"body": "PogChamp Kappa"}, ...}, ...]
//
// The ctx.Done() channel is respected between file read and window processing.
func (d *ChatActivityDetector) DetectHighlights(ctx context.Context, chatJSONPath string) ([]ChatHighlight, error) {
	d.logger.Info("chat detect started", "path", chatJSONPath)

	data, err := os.ReadFile(chatJSONPath)
	if err != nil {
		d.logger.Error("chat detect: read file failed", "err", err, "path", chatJSONPath)
		return nil, fmt.Errorf("chat_activity_detector: read file: %w", err)
	}

	var messages []twitchChatMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		d.logger.Error("chat detect: parse JSON failed", "err", err)
		return nil, fmt.Errorf("chat_activity_detector: parse json: %w", err)
	}

	if len(messages) == 0 {
		d.logger.Warn("chat detect: no messages found")
		return nil, nil
	}

	// Sort by time (TwitchDownloaderCLI exports in order, but be defensive).
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].ContentOffsetSeconds < messages[j].ContentOffsetSeconds
	})

	maxTime := messages[len(messages)-1].ContentOffsetSeconds

	// Build sliding windows.
	type windowDensity struct {
		startSec float64
		density  float64
		reason   string
	}
	var windows []windowDensity

	for winStart := 0.0; winStart < maxTime; winStart += chatWindowStep {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		winEnd := winStart + chatWindowSec
		var (
			total       int
			emoteCount  int
			capsCount   int
		)

		for _, msg := range messages {
			t := msg.ContentOffsetSeconds
			if t < winStart {
				continue
			}
			if t >= winEnd {
				break
			}
			total++
			body := msg.Message.Body
			if hasEmoteSpike(body) {
				emoteCount++
			}
			if hasCapsExcitement(body) {
				capsCount++
			}
		}

		if total == 0 {
			continue
		}

		density := float64(emoteCount+capsCount) / float64(total)
		reason := "emote_spike"
		if capsCount > emoteCount {
			reason = "caps_excitement"
		}

		windows = append(windows, windowDensity{
			startSec: winStart,
			density:  density,
			reason:   reason,
		})
	}

	if len(windows) == 0 {
		d.logger.Info("chat detect done: no active windows")
		return nil, nil
	}

	// Normalise densities.
	maxDensity := 0.0
	for _, w := range windows {
		if w.density > maxDensity {
			maxDensity = w.density
		}
	}

	// Collect windows above threshold.
	type candidate struct {
		startSec float64
		endSec   float64
		score    float64
		reason   string
	}
	var candidates []candidate
	for _, w := range windows {
		normDensity := 0.0
		if maxDensity > 0 {
			normDensity = w.density / maxDensity
		}
		if normDensity < chatDensityThreshold {
			continue
		}
		candidates = append(candidates, candidate{
			startSec: w.startSec,
			endSec:   w.startSec + chatWindowSec,
			score:    clampScore(normDensity),
			reason:   w.reason,
		})
	}

	// Merge adjacent candidates.
	var highlights []ChatHighlight
	for i, c := range candidates {
		if i == 0 {
			highlights = append(highlights, ChatHighlight{
				StartSec: c.startSec,
				EndSec:   c.endSec,
				Score:    c.score,
				Reason:   c.reason,
			})
			continue
		}
		last := &highlights[len(highlights)-1]
		if c.startSec-last.EndSec <= chatMergeGapSec {
			// Extend the last highlight.
			last.EndSec = c.endSec
			if c.score > last.Score {
				last.Score = c.score
				last.Reason = c.reason
			}
		} else {
			highlights = append(highlights, ChatHighlight{
				StartSec: c.startSec,
				EndSec:   c.endSec,
				Score:    c.score,
				Reason:   c.reason,
			})
		}
	}

	d.logger.Info("chat detect done", "highlights", len(highlights))
	return highlights, nil
}

// hasEmoteSpike returns true if the message body contains words that look like
// Twitch emotes (ALL_CAPS word, length > 2, all alphabetic).
func hasEmoteSpike(body string) bool {
	for _, word := range strings.Fields(body) {
		if len(word) <= 2 {
			continue
		}
		upper := true
		for _, r := range word {
			if !unicode.IsLetter(r) {
				upper = false
				break
			}
			if unicode.IsLower(r) {
				upper = false
				break
			}
		}
		if upper {
			return true
		}
	}
	return false
}

// hasCapsExcitement returns true when more than 50% of the message's letter
// characters are uppercase (excludes punctuation and spaces from the count).
func hasCapsExcitement(body string) bool {
	var total, caps int
	for _, r := range body {
		if unicode.IsLetter(r) {
			total++
			if unicode.IsUpper(r) {
				caps++
			}
		}
	}
	if total == 0 {
		return false
	}
	return float64(caps)/float64(total) > 0.5
}
