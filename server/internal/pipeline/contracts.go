package pipeline

// PhaseContract declares the fields a pipeline phase requires as inputs and
// produces as outputs. Downstream phases call ValidatePhaseInputs to enforce
// these contracts at runtime before starting any external process.
type PhaseContract struct {
	// ID is the PhaseID constant (e.g. PhaseDownload, PhaseTranscribe).
	ID string

	// EntryState is the RunState the pipeline is in when this phase begins.
	EntryState string

	// ExitSuccess is the RunState written on successful completion.
	ExitSuccess string

	// ExitError is the RunState written on failure.
	ExitError string

	// ExitCancel is the RunState written when the phase is cancelled.
	ExitCancel string

	// RequiredInputs lists the database column names that must be non-empty
	// before this phase may execute.
	RequiredInputs []string

	// ProducedOutputs lists the database column names this phase writes before
	// transitioning to ExitSuccess.
	ProducedOutputs []string
}

// DownloadContract is the canonical contract for the download phase.
// Any phase that follows WAITING_MODE may depend on all ProducedOutputs.
var DownloadContract = PhaseContract{
	ID:              PhaseDownload,
	EntryState:      StateExecuting,
	ExitSuccess:     StateWaitingMode,
	ExitError:       StateError,
	ExitCancel:      StateCancelled,
	RequiredInputs:  []string{"url"},
	ProducedOutputs: []string{"video_path", "video_title", "duration_sec"},
}
