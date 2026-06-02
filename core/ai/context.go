package ai

// Confidence indicates how strongly the diagnosis is supported by verified evidence.
type Confidence string

const (
	ConfidenceHigh   Confidence = "High"
	ConfidenceMedium Confidence = "Medium"
	ConfidenceLow    Confidence = "Low"
)

// DiagnosticContext holds structured diagnostic data for a failed pipeline stage.
// Diagnosis must be driven by verified evidence in this struct, not raw log keywords alone.
type DiagnosticContext struct {
	Command       string
	Stage         string
	Error         string
	ExitCode      int
	RecentLogs    []string
	SkippedStages []string
	FailedStage   string // Jenkins-specific failed stage name, if detected
	FinalStatus   string // e.g. FAILURE, ABORTED

	// Verified environment state (populated by CollectEvidence)
	PodStatus         string
	PodRestarts       int
	PodNamespace      string
	PodName           string
	ServiceStatus     string
	ServiceExists     bool
	ServiceNamespace  string
	RegistryStatus    string
	RegistryReachable bool
	DockerDaemonOK    bool
	JenkinsStatus     string
	TunnelEstablished bool
	PortForwardFailed bool

	Evidence []string
}

// FailureContext is an alias for backward compatibility with existing call sites.
type FailureContext = DiagnosticContext
