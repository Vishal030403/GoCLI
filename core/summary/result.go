package summary

import (
	"fmt"
	"strings"
	"time"
)

func buildExecutionResult(state ExecutionState) string {
	status := "SUCCESS"
	if !state.Success {
		status = "FAILED"
	}
	errors := errorCount(state)
	warnings := warningStageCount(state)

	line := fmt.Sprintf("%s · %s · warnings:%d · errors:%d",
		status, formatDuration(state), warnings, errors)
	if !state.Success && state.FailedStage != "" {
		line += " · failed:" + displayStageName(state.FailedStage)
	}
	return line
}

func errorCount(state ExecutionState) int {
	n := len(state.Errors)
	for _, s := range state.Stages {
		if s.Status == StageFailed {
			n++
		}
	}
	return n
}

func formatDuration(state ExecutionState) string {
	d := state.Duration
	if d == 0 && !state.EndTime.IsZero() && !state.StartTime.IsZero() {
		d = state.EndTime.Sub(state.StartTime)
	}
	if d <= 0 {
		return "unknown"
	}
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func formatTunnelMetrics(state ExecutionState) string {
	if state.Tunnel == nil {
		return ""
	}
	t := state.Tunnel
	dur := formatDuration(state)
	if t.Duration > 0 {
		dur = t.Duration.Round(time.Second).String()
	}
	outcome := t.Outcome
	if outcome == "" {
		outcome = "ended"
	}
	return fmt.Sprintf("%s/%s · localhost:%s→%s · %s · %d requests · %s",
		t.AppName, t.Namespace, t.LocalPort, t.TargetPort, dur, t.RequestsForwarded, outcome)
}

func formatInfrastructureBrief(state ExecutionState) string {
	if len(state.Infrastructure) == 0 {
		return ""
	}
	parts := make([]string, 0, len(state.Infrastructure))
	for i, item := range state.Infrastructure {
		if i >= 4 {
			break
		}
		parts = append(parts, item.Name)
	}
	return strings.Join(parts, ", ")
}
