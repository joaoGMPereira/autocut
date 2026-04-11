package uploader

// UploadStrategy controls concurrency for batch uploads.
type UploadStrategy struct {
	parallel bool
	sem      chan struct{} // semaphore for parallel slots
}

// NewUploadStrategy creates a strategy from settings.
// strategy: "sequential" | "parallel"
// count: 1-5 (number of parallel slots, only meaningful for "parallel" strategy)
func NewUploadStrategy(strategy string, count int) *UploadStrategy {
	if count < 1 {
		count = 1
	}
	if count > 5 {
		count = 5
	}
	s := &UploadStrategy{parallel: strategy == "parallel"}
	// FP-014 fix: sequential mode must always use capacity=1 regardless of count,
	// so that Acquire() blocks until the previous upload finishes.
	if strategy == "sequential" {
		s.sem = make(chan struct{}, 1)
	} else {
		s.sem = make(chan struct{}, count)
	}
	return s
}

// Acquire acquires a slot. In sequential mode, this is always 1 slot.
func (s *UploadStrategy) Acquire() {
	s.sem <- struct{}{}
}

// Release releases a slot.
func (s *UploadStrategy) Release() {
	<-s.sem
}

// IsParallel returns true if parallel strategy is active.
func (s *UploadStrategy) IsParallel() bool {
	return s.parallel
}
