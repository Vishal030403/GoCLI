package ai

// FailureContext holds structured diagnostic data for a failed pipeline stage.
// Logs are trimmed before any AI call — never send full log dumps to Gemini.
type FailureContext struct {
	Command       string
	Stage         string
	Error         string
	ExitCode      int
	RecentLogs    []string
	SkippedStages []string
	FailedStage   string // Jenkins-specific failed stage name, if detected
	FinalStatus   string // e.g. FAILURE, ABORTED
}

// WarningItem represents a non-fatal policy or validation finding.
type WarningItem struct {
	PolicyName string
	Category   string
	Message    string
	Findings   []string
}
