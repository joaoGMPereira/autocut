package pipeline

import (
	"github.com/joaoGMPereira/autocut/server/internal/hub"
)

// publishStateChanged notifies SSE clients that the run state has changed.
func publishStateChanged(h *hub.SSEHub, runID int64, state RunState) {
	h.Publish(PipelineJobID(runID), hub.SSEEvent{
		Type: SSEEventStateChanged,
		Data: map[string]interface{}{
			"run_id": runID,
			"state":  state,
		},
	})
}

// publishGateOpened notifies SSE clients that a gate state has been reached
// and user action is required.
func publishGateOpened(h *hub.SSEHub, runID int64, state RunState) {
	h.Publish(PipelineJobID(runID), hub.SSEEvent{
		Type: SSEEventGateOpened,
		Data: map[string]interface{}{
			"run_id": runID,
			"state":  state,
		},
	})
}

// publishPhaseProgress notifies SSE clients of sub-phase progress within a
// compute state (EXECUTING / GENERATING_CLIPS / UPLOADING).
func publishPhaseProgress(h *hub.SSEHub, runID int64, phase ExecutePhase, percentDone float64, extra map[string]interface{}) {
	data := map[string]interface{}{
		"run_id":       runID,
		"phase":        phase,
		"percent_done": percentDone,
	}
	for k, v := range extra {
		data[k] = v
	}
	h.Publish(PipelineJobID(runID), hub.SSEEvent{
		Type: SSEEventPhaseProgress,
		Data: data,
	})
}

// publishDone notifies SSE clients that the run has completed successfully.
func publishDone(h *hub.SSEHub, runID int64) {
	h.Publish(PipelineJobID(runID), hub.SSEEvent{
		Type: SSEEventDone,
		Data: map[string]interface{}{
			"run_id": runID,
			"state":  StateDone,
		},
	})
}

// publishError notifies SSE clients that the run has failed.
func publishError(h *hub.SSEHub, runID int64, errMsg string) {
	h.Publish(PipelineJobID(runID), hub.SSEEvent{
		Type: SSEEventError,
		Data: map[string]interface{}{
			"run_id":  runID,
			"state":   StateError,
			"message": errMsg,
		},
	})
}

// publishCancelled notifies SSE clients that the run was cancelled.
func publishCancelled(h *hub.SSEHub, runID int64) {
	h.Publish(PipelineJobID(runID), hub.SSEEvent{
		Type: SSEEventCancelled,
		Data: map[string]interface{}{
			"run_id": runID,
			"state":  StateCancelled,
		},
	})
}
