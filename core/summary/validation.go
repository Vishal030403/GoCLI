package summary

import (
	"strings"
)

// sanitizeReport removes hallucinated warnings/failures not present in execution state.
func sanitizeReport(state ExecutionState, r SummaryReport) SummaryReport {
	warnCount := warningStageCount(state)
	hasWarnings := warnCount > 0
	hasFailures := !state.Success || state.FailedStage != ""

	if !hasWarnings {
		r = stripWarningLanguage(r)
	}
	if !hasFailures {
		r.FailedStage = ""
		r.RecoverySteps = ""
		if state.Success {
			r.SuccessfulStages = ""
			r.SkippedStages = ""
			r.InfrastructureState = ""
		}
	}
	if len(SkippedStageNames(state)) == 0 {
		r.SkippedStages = ""
	}
	return r
}

func stripWarningLanguage(r SummaryReport) SummaryReport {
	fields := []*string{
		&r.ExecutionOverview, &r.Infrastructure, &r.ValidationResults,
		&r.PipelineStages, &r.PipelineOutcome, &r.KeyLearnings, &r.Recommendations,
		&r.OverallStatus, &r.DeveloperNotes,
	}
	for _, f := range fields {
		*f = removeWarningPhrases(*f)
	}
	return r
}

func removeWarningPhrases(s string) string {
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	if !strings.Contains(lower, "warn") {
		return s
	}
	var kept []string
	for _, line := range strings.Split(s, "\n") {
		l := strings.ToLower(line)
		if strings.Contains(l, "warning") || strings.Contains(l, "warn ") || strings.Contains(l, "with warnings") {
			continue
		}
		kept = append(kept, line)
	}
	out := strings.TrimSpace(strings.Join(kept, "\n"))
	if out == "" {
		return s
	}
	return out
}
