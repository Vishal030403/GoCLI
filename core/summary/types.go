package summary

import "time"

// StageStatus represents the outcome of a pipeline stage.
type StageStatus string

const (
	StageSuccess StageStatus = "SUCCESS"
	StageFailed  StageStatus = "FAILED"
	StageSkipped StageStatus = "SKIPPED"
	StageWarning StageStatus = "WARNING"
)

// ExecutionState is structured runtime data for summary generation.
type ExecutionState struct {
	Command        string
	Success        bool
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
	Stages         []StageRecord
	FailedStage    string
	Warnings       []string
	Errors         []string
	Infrastructure []InfrastructureItem
	Metadata       map[string]string
}

// StageRecord captures one named stage outcome.
type StageRecord struct {
	Name    string
	Status  StageStatus
	Message string
}

// InfrastructureItem describes a resource created or affected.
type InfrastructureItem struct {
	Name   string
	Detail string
}

// SummaryReport is the rendered summary (terminal + markdown).
type SummaryReport struct {
	ExecutionOverview   string
	Infrastructure      string
	ValidationResults   string
	PipelineStages      string
	KeyLearnings        string
	Recommendations     string
	SuccessfulStages    string
	FailedStage         string
	SkippedStages       string
	InfrastructureState string
	RecoverySteps       string
	OverallStatus       string
	RawMarkdown         string
}
