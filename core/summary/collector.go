package summary

import "strings"

// StructuredPayload builds JSON for Gemini — structured state only, no logs.
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

	warnCount := warningStageCount(state)
	errCount := len(state.Errors)
	for _, s := range state.Stages {
		if s.Status == StageFailed {
			errCount++
		}
	}

	payload := map[string]interface{}{
		"command":         commandShort(state.Command),
		"success":         state.Success,
		"duration":        formatDuration(state),
		"failed_stage":    state.FailedStage,
		"stages":          stages,
		"warnings":        state.Warnings,
		"warnings_count":  warnCount,
		"errors":          state.Errors,
		"errors_count":    errCount,
		"infrastructure":  infra,
		"metadata":        state.Metadata,
		"rules": map[string]string{
			"status_meaning": "SUCCESS means completed without error. Do not describe SUCCESS stages as warnings.",
			"warnings_rule":  "If warnings_count is 0, do not mention warnings anywhere.",
			"skipped_rule":   "Only list skipped stages present in stages with status SKIPPED.",
		},
	}

	if state.Tunnel != nil {
		payload["tunnel"] = map[string]interface{}{
			"app_name":            state.Tunnel.AppName,
			"namespace":           state.Tunnel.Namespace,
			"local_port":          state.Tunnel.LocalPort,
			"target_port":         state.Tunnel.TargetPort,
			"duration":            state.Tunnel.Duration.String(),
			"requests_forwarded":  state.Tunnel.RequestsForwarded,
			"outcome":             state.Tunnel.Outcome,
		}
	}

	return payload
}

func commandShort(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, "pipeline ") {
		return strings.TrimPrefix(cmd, "pipeline ")
	}
	return cmd
}
