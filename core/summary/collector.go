package summary

import (
	"strings"
	"time"
)

// StructuredPayload builds JSON-friendly context for Gemini (no raw logs).
func StructuredPayload(state ExecutionState) map[string]interface{} {
	stages := make([]map[string]string, 0, len(state.Stages))
	for _, s := range state.Stages {
		stages = append(stages, map[string]string{
			"name":    s.Name,
			"status":  string(s.Status),
			"message": s.Message,
		})
	}

	infra := make([]map[string]string, 0, len(state.Infrastructure))
	for _, i := range state.Infrastructure {
		infra = append(infra, map[string]string{
			"name":   i.Name,
			"detail": i.Detail,
		})
	}

	duration := state.Duration.String()
	if state.Duration == 0 && !state.EndTime.IsZero() {
		duration = state.EndTime.Sub(state.StartTime).Round(time.Second).String()
	}

	return map[string]interface{}{
		"command":        commandShort(state.Command),
		"success":        state.Success,
		"duration":       duration,
		"failed_stage":   state.FailedStage,
		"stages":         stages,
		"warnings":       state.Warnings,
		"errors":         state.Errors,
		"infrastructure": infra,
		"metadata":       state.Metadata,
	}
}

func commandShort(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, "pipeline ") {
		return strings.TrimPrefix(cmd, "pipeline ")
	}
	return cmd
}

// SuccessfulStageNames returns stages that completed successfully.
func SuccessfulStageNames(state ExecutionState) []string {
	var out []string
	for _, s := range state.Stages {
		if s.Status == StageSuccess {
			out = append(out, s.Name)
		}
	}
	return out
}

// SkippedStageNames returns stages marked skipped.
func SkippedStageNames(state ExecutionState) []string {
	var out []string
	for _, s := range state.Stages {
		if s.Status == StageSkipped {
			out = append(out, s.Name)
		}
	}
	return out
}
