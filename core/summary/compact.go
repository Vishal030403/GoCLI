package summary

import "strings"

const (
	maxLine   = 160
	maxBullet = 4
)

func clamp(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func clampLines(s string, maxLines int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

// compactForTerminal shortens report fields shown in the CLI.
func compactForTerminal(r SummaryReport) SummaryReport {
	r.ExecutionOverview = clamp(r.ExecutionOverview, maxLine)
	r.Infrastructure = clampLines(r.Infrastructure, maxBullet)
	r.ValidationResults = clamp(r.ValidationResults, maxLine)
	r.PipelineStages = clampLines(r.PipelineStages, maxBullet)
	r.PipelineOutcome = clamp(r.PipelineOutcome, maxLine)
	r.KeyLearnings = clamp(r.KeyLearnings, maxLine)
	r.Recommendations = clamp(r.Recommendations, maxLine)
	r.SuccessfulStages = clamp(r.SuccessfulStages, maxLine)
	r.FailedStage = clamp(r.FailedStage, 80)
	r.SkippedStages = clamp(r.SkippedStages, maxLine)
	r.InfrastructureState = clampLines(r.InfrastructureState, 3)
	r.RecoverySteps = clampLines(r.RecoverySteps, 3)
	r.ProjectDetection = clamp(r.ProjectDetection, 80)
	r.GeneratedFiles = clamp(r.GeneratedFiles, maxLine)
	r.NextSteps = clamp(r.NextSteps, maxLine)
	r.TunnelOverview = clamp(r.TunnelOverview, maxLine)
	r.TunnelMetrics = clamp(r.TunnelMetrics, maxLine)
	r.SessionOutcome = clamp(r.SessionOutcome, maxLine)
	r.CleanupOverview = clamp(r.CleanupOverview, maxLine)
	r.ResourcesRemoved = clampLines(r.ResourcesRemoved, 3)
	r.EnvironmentState = clamp(r.EnvironmentState, maxLine)
	return r
}
