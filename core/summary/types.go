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

// TunnelSummary holds port-forward session metrics.
type TunnelSummary struct {
	AppName           string
	Namespace         string
	LocalPort         string
	TargetPort        string
	StartTime         time.Time
	EndTime           time.Time
	Duration          time.Duration
	RequestsForwarded int
	Outcome           string
}

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
	Tunnel         *TunnelSummary
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
	ExecutionResult     string
	ExecutionOverview   string
	Infrastructure      string
	ValidationResults   string
	PipelineStages      string
	PipelineOutcome     string
	KeyLearnings        string
	Recommendations     string
	SuccessfulStages    string
	FailedStage         string
	SkippedStages       string
	InfrastructureState string
	RecoverySteps       string
	OverallStatus       string
	// init
	ProjectDetection string
	GeneratedFiles   string
	NextSteps        string
	// tunnel
	TunnelOverview string
	TunnelMetrics  string
	SessionOutcome string
	// destroy-ci
	CleanupOverview  string
	ResourcesRemoved string
	ClusterStatus    string
	RegistryStatus   string
	JenkinsStatus    string
	EnvironmentState string
	// markdown-only extras
	DeveloperNotes string
	RawMarkdown    string
}
