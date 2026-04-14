package pipeline

import (
	"fmt"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ValidatePhaseInputs checks that all fields required by the given phase are
// non-empty/non-zero on the run record. Returns a descriptive error if any
// required field is missing; returns nil if all inputs are present.
//
// Phase entry points MUST call this before starting any external process —
// if it returns an error, transition to ERROR with the returned message.
func ValidatePhaseInputs(run *database.PipelineRun, phaseID string) error {
	switch phaseID {
	case PhaseDownload:
		if run.URL == "" {
			return fmt.Errorf("phase %q requires url (got empty)", phaseID)
		}
	case PhaseTranscribe, PhaseAnalyze, PhaseSegment, PhaseCut, PhaseShorts, PhaseThumbnail, PhaseUpload:
		if run.VideoPath == "" {
			return fmt.Errorf("phase %q requires video_path (got empty)", phaseID)
		}
		if run.VideoTitle == "" {
			return fmt.Errorf("phase %q requires video_title (got empty)", phaseID)
		}
		if run.DurationSec <= 0 {
			return fmt.Errorf("phase %q requires duration_sec > 0 (got %d)", phaseID, run.DurationSec)
		}
	}
	return nil
}
