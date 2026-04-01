package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// persistStepOutput reads the current step_outputs_json for the given pipeline
// run (identified by sessionID), merges the new stepName → output entry, and
// writes it back. It also creates/updates a pipeline_run_steps record.
// Best-effort: errors are logged but not propagated.
func persistStepOutput(repo *database.PipelineRunRepo, sessionID string, stepName string, output interface{}) {
	if repo == nil || sessionID == "" {
		return
	}

	runID, err := strconv.ParseInt(sessionID, 10, 64)
	if err != nil {
		slog.Warn("persistStepOutput: invalid session_id", "session_id", sessionID, "err", err)
		return
	}

	ctx := context.Background()

	run, err := repo.GetByID(ctx, runID)
	if err != nil {
		slog.Warn("persistStepOutput: get run failed", "run_id", runID, "err", err)
		return
	}

	var outputs map[string]interface{}
	if err := json.Unmarshal([]byte(run.StepOutputsJSON), &outputs); err != nil {
		outputs = map[string]interface{}{}
	}

	outputs[stepName] = output

	merged, err := json.Marshal(outputs)
	if err != nil {
		slog.Warn("persistStepOutput: marshal failed", "run_id", runID, "err", err)
		return
	}

	if err := repo.UpdateStepOutput(ctx, runID, string(merged)); err != nil {
		slog.Warn("persistStepOutput: update failed", "run_id", runID, "err", err)
	}

	// Also create/update a step record in pipeline_run_steps
	step := &database.PipelineRunStep{
		RunID:  runID,
		Step:   stepName,
		Status: "done",
	}
	if _, err := repo.CreateStep(ctx, step); err != nil {
		slog.Warn("persistStepOutput: create step failed", "run_id", runID, "step", stepName, "err", err)
	}
}
