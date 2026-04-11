package ai

import (
	"log/slog"
	"sort"
)

// Strategy identifies the detection method that produced a highlight.
type Strategy string

const (
	StrategyAI     Strategy = "ai"
	StrategyAudio  Strategy = "audio"
	StrategyVisual Strategy = "visual"
	StrategyChat   Strategy = "chat"
)

// UnifiedHighlight is the combined output of the HighlightAggregator, merging
// signals from one or more detection strategies into a single scored entry.
type UnifiedHighlight struct {
	StartSec   float64            `json:"start_sec"`
	EndSec     float64            `json:"end_sec"`
	Confidence float64            `json:"confidence"`           // weighted average, 0.0–1.0
	Strategies []Strategy         `json:"strategies"`           // which strategies detected this
	Scores     map[string]float64 `json:"scores"`               // per-strategy raw score
	Type       string             `json:"type"`                 // always "highlight"
}

// HighlightAggregator merges highlights from multiple detection strategies
// into a unified list with weighted confidence scoring.
type HighlightAggregator struct {
	Weights map[Strategy]float64
	logger  *slog.Logger
}

// defaultWeights returns the default strategy weight map.
// Total weights sum to 1.0.
func defaultWeights() map[Strategy]float64 {
	return map[Strategy]float64{
		StrategyAI:     0.4,
		StrategyAudio:  0.3,
		StrategyVisual: 0.2,
		StrategyChat:   0.1,
	}
}

// NewHighlightAggregator creates an aggregator with default weights
// (AI=0.4, audio=0.3, visual=0.2, chat=0.1).
func NewHighlightAggregator() *HighlightAggregator {
	return &HighlightAggregator{
		Weights: defaultWeights(),
		logger:  slog.With("component", "highlight_aggregator"),
	}
}

// interval is a strategy-tagged time range with a normalised score.
type interval struct {
	startSec float64
	endSec   float64
	score    float64
	strategy Strategy
}

// Aggregate combines highlights from multiple strategies into a unified list.
//
// Pipeline:
//  1. Convert all source highlights into a common []interval list.
//  2. Sort by start time.
//  3. Merge intervals that overlap by more than 50% of the shorter interval.
//  4. Compute confidence as the weighted average of strategy scores present.
//  5. Return sorted by confidence descending.
//
// The strategies parameter controls which strategy inputs are active. If a
// strategy is not in the list its highlights are ignored even if non-nil data
// is supplied.
func (a *HighlightAggregator) Aggregate(
	aiHighlights []Highlight,
	audioHighlights []AudioHighlight,
	visualChanges []SceneChange,
	chatHighlights []ChatHighlight,
	strategies []Strategy,
) []UnifiedHighlight {
	active := make(map[Strategy]bool, len(strategies))
	for _, s := range strategies {
		active[s] = true
	}

	var intervals []interval

	if active[StrategyAI] {
		for _, h := range aiHighlights {
			intervals = append(intervals, interval{
				startSec: h.StartSec,
				endSec:   h.EndSec,
				score:    h.Score,
				strategy: StrategyAI,
			})
		}
	}

	if active[StrategyAudio] {
		for _, h := range audioHighlights {
			intervals = append(intervals, interval{
				startSec: h.StartSec,
				endSec:   h.EndSec,
				score:    h.Score,
				strategy: StrategyAudio,
			})
		}
	}

	if active[StrategyVisual] {
		const sceneWindowSec = 5.0
		for _, sc := range visualChanges {
			intervals = append(intervals, interval{
				startSec: sc.TimeSec - sceneWindowSec/2,
				endSec:   sc.TimeSec + sceneWindowSec/2,
				score:    sc.Score,
				strategy: StrategyVisual,
			})
		}
	}

	if active[StrategyChat] {
		for _, h := range chatHighlights {
			intervals = append(intervals, interval{
				startSec: h.StartSec,
				endSec:   h.EndSec,
				score:    h.Score,
				strategy: StrategyChat,
			})
		}
	}

	if len(intervals) == 0 {
		return nil
	}

	// Sort by start time for merge pass.
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].startSec < intervals[j].startSec
	})

	// Merge overlapping intervals. Two intervals overlap by > 50% of the
	// shorter one when the overlap length exceeds half the shorter duration.
	merged := a.mergeIntervals(intervals)
	a.logger.Debug("aggregator merge done", "input_intervals", len(intervals), "merged", len(merged))

	// Build UnifiedHighlight entries.
	result := make([]UnifiedHighlight, 0, len(merged))
	for _, group := range merged {
		uh := a.buildUnified(group)
		result = append(result, uh)
	}

	// Sort by confidence descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Confidence > result[j].Confidence
	})

	a.logger.Info("aggregator done", "unified_highlights", len(result))
	return result
}

// mergedGroup holds a set of intervals that were merged together.
type mergedGroup struct {
	startSec  float64
	endSec    float64
	intervals []interval
}

// mergeIntervals merges intervals that overlap by >50% of the shorter one.
func (a *HighlightAggregator) mergeIntervals(intervals []interval) []mergedGroup {
	if len(intervals) == 0 {
		return nil
	}

	groups := []mergedGroup{
		{
			startSec:  intervals[0].startSec,
			endSec:    intervals[0].endSec,
			intervals: []interval{intervals[0]},
		},
	}

	for _, iv := range intervals[1:] {
		last := &groups[len(groups)-1]
		overlapStart := max64(last.startSec, iv.startSec)
		overlapEnd := min64(last.endSec, iv.endSec)
		overlapLen := overlapEnd - overlapStart

		shortLen := min64(last.endSec-last.startSec, iv.endSec-iv.startSec)
		if shortLen <= 0 {
			shortLen = 0.001
		}

		if overlapLen > 0 && overlapLen/shortLen > 0.5 {
			// Merge: expand the group's time range.
			if iv.endSec > last.endSec {
				last.endSec = iv.endSec
			}
			last.intervals = append(last.intervals, iv)
		} else {
			groups = append(groups, mergedGroup{
				startSec:  iv.startSec,
				endSec:    iv.endSec,
				intervals: []interval{iv},
			})
		}
	}

	return groups
}

// buildUnified converts a mergedGroup into a UnifiedHighlight with weighted
// confidence scoring. Strategy scores are averaged per strategy first, then
// the weighted average across strategies is computed.
func (a *HighlightAggregator) buildUnified(group mergedGroup) UnifiedHighlight {
	// Collect scores per strategy (average if multiple intervals of same strategy).
	stratScoreSum := make(map[Strategy]float64)
	stratCount := make(map[Strategy]int)
	for _, iv := range group.intervals {
		stratScoreSum[iv.strategy] += iv.score
		stratCount[iv.strategy]++
	}

	scores := make(map[string]float64, len(stratScoreSum))
	for s, sum := range stratScoreSum {
		scores[string(s)] = sum / float64(stratCount[s])
	}

	weights := a.Weights
	if weights == nil {
		weights = defaultWeights()
	}

	var weightSum, weightedScore float64
	var strategyList []Strategy
	for s, score := range scores {
		st := Strategy(s)
		w := weights[st]
		weightedScore += score * w
		weightSum += w
		strategyList = append(strategyList, st)
	}

	confidence := 0.0
	if weightSum > 0 {
		confidence = clampScore(weightedScore / weightSum)
	}

	// Sort strategy list for deterministic output.
	sort.Slice(strategyList, func(i, j int) bool {
		return string(strategyList[i]) < string(strategyList[j])
	})

	return UnifiedHighlight{
		StartSec:   group.startSec,
		EndSec:     group.endSec,
		Confidence: confidence,
		Strategies: strategyList,
		Scores:     scores,
		Type:       "highlight",
	}
}

// max64 returns the larger of two float64 values.
func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// min64 returns the smaller of two float64 values.
func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
