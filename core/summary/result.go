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
	dur := formatDuration(state)
	warnings := warningStageCount(state)
	errors := len(state.Errors)
	for _, s := range state.Stages {
		if s.Status == StageFailed {
			errors++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Status: %s\n", status)
	fmt.Fprintf(&b, "Duration: %s\n", dur)
	fmt.Fprintf(&b, "Warnings: %d\n", warnings)
	fmt.Fprintf(&b, "Errors: %d", errors)
	if !state.Success && state.FailedStage != "" {
		b.WriteString("\nFailed Stage: " + displayStageName(state.FailedStage))
	}
	return b.String()
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
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
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
	var b strings.Builder
	fmt.Fprintf(&b, "Application: %s\n", t.AppName)
	fmt.Fprintf(&b, "Namespace: %s\n", t.Namespace)
	fmt.Fprintf(&b, "Local Port: %s\n", t.LocalPort)
	fmt.Fprintf(&b, "Pod/Service Port: %s\n", t.TargetPort)
	fmt.Fprintf(&b, "Duration: %s\n", dur)
	fmt.Fprintf(&b, "Requests Forwarded: %d\n", t.RequestsForwarded)
	if t.Outcome != "" {
		fmt.Fprintf(&b, "Session Outcome: %s", t.Outcome)
	}
	return strings.TrimSpace(b.String())
}
