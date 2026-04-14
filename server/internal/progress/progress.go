package progress

// Event is emitted by any component that reports progress.
// Percent == -1 signals indeterminate progress (spinner); 0–100 signals known progress (bar).
type Event struct {
	Stage   string  `json:"stage"`
	Message string  `json:"message,omitempty"`
	Percent float64 `json:"percent"`
}

// Reporter receives progress events. Implementations must be safe for concurrent use.
type Reporter interface {
	Report(jobID string, event Event)
}

// NoopReporter discards all events. Safe for concurrent use. Use as the default Reporter.
type NoopReporter struct{}

func (NoopReporter) Report(_ string, _ Event) {}
