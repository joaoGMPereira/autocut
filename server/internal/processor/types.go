package processor

import "time"

// SpeedZone defines a time range within a video where a specific playback speed is applied.
// Kotlin ref: SpeedZone model in com.clipper.core.models.
type SpeedZone struct {
	Start time.Duration
	End   time.Duration
	Speed float64
}

// OptimizeResult holds statistics produced after a video optimization run.
// Kotlin ref: OptimizationResult in VideoOptimizerProcessor.
type OptimizeResult struct {
	OriginalDuration time.Duration
	FinalDuration    time.Duration
	RemovedSilences  int
}
