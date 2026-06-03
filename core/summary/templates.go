package summary

type sectionID string

const (
	secResult          sectionID = "execution_result"
	secOverview        sectionID = "execution_overview"
	secInfra           sectionID = "infrastructure"
	secValidation      sectionID = "validation"
	secStages          sectionID = "pipeline_stages"
	secOutcome         sectionID = "pipeline_outcome"
	secLearnings       sectionID = "key_learnings"
	secRecommendations sectionID = "recommendations"
	secSuccessStages   sectionID = "successful_stages"
	secFailed          sectionID = "failed_stage"
	secSkipped         sectionID = "skipped_stages"
	secInfraState      sectionID = "infrastructure_state"
	secRecovery        sectionID = "recovery"
	secProjectDetect   sectionID = "project_detection"
	secGeneratedFiles  sectionID = "generated_files"
	secNextSteps       sectionID = "next_steps"
	secTunnelOverview  sectionID = "tunnel_overview"
	secTunnelMetrics   sectionID = "tunnel_metrics"
	secSessionOutcome  sectionID = "session_outcome"
	secCleanup         sectionID = "cleanup_overview"
	secResourcesRemoved sectionID = "resources_removed"
	secCluster         sectionID = "cluster_status"
	secRegistry        sectionID = "registry_status"
	secJenkins         sectionID = "jenkins_status"
	secEnvironment     sectionID = "environment_state"
)

type templateLayout struct {
	terminalSuccess []sectionID
	terminalFailure []sectionID
	markdownExtra   []sectionID
}

var commandTemplates = map[string]templateLayout{
	"init": {
		terminalSuccess: []sectionID{secResult, secOverview, secNextSteps},
		terminalFailure: []sectionID{secResult, secFailed, secRecovery},
		markdownExtra:   []sectionID{secStages, secGeneratedFiles, secLearnings},
	},
	"prep-ci": {
		terminalSuccess: []sectionID{secResult, secOutcome, secInfra, secRecommendations},
		terminalFailure: []sectionID{secResult, secFailed, secSkipped, secRecovery},
		markdownExtra:   []sectionID{secStages, secValidation, secLearnings},
	},
	"tunnel": {
		terminalSuccess: []sectionID{secResult, secTunnelMetrics, secRecommendations},
		terminalFailure: []sectionID{secResult, secFailed, secRecovery},
		markdownExtra:   []sectionID{secTunnelOverview, secSessionOutcome},
	},
	"destroy-ci": {
		terminalSuccess: []sectionID{secResult, secCleanup, secRecommendations},
		terminalFailure: []sectionID{secResult, secFailed, secRecovery},
		markdownExtra:   []sectionID{secResourcesRemoved, secEnvironment},
	},
}

func layoutFor(state ExecutionState) templateLayout {
	cmd := commandShort(state.Command)
	if t, ok := commandTemplates[cmd]; ok {
		return t
	}
	return templateLayout{
		terminalSuccess: []sectionID{secResult, secOutcome, secRecommendations},
		terminalFailure: []sectionID{secResult, secFailed, secRecovery},
		markdownExtra:   []sectionID{secStages},
	}
}

func sectionTitle(id sectionID) string {
	switch id {
	case secResult:
		return "Result"
	case secOverview:
		return "Overview"
	case secInfra:
		return "Infrastructure"
	case secValidation:
		return "Validation"
	case secStages:
		return "Stages"
	case secOutcome:
		return "Outcome"
	case secLearnings:
		return "Learnings"
	case secRecommendations:
		return "Next"
	case secSuccessStages:
		return "Completed"
	case secFailed:
		return "Failed"
	case secSkipped:
		return "Skipped"
	case secInfraState:
		return "Infra State"
	case secRecovery:
		return "Recovery"
	case secProjectDetect:
		return "Framework"
	case secGeneratedFiles:
		return "Files"
	case secNextSteps:
		return "Next"
	case secTunnelOverview:
		return "Tunnel"
	case secTunnelMetrics:
		return "Tunnel"
	case secSessionOutcome:
		return "Session"
	case secCleanup:
		return "Cleanup"
	case secResourcesRemoved:
		return "Removed"
	case secEnvironment:
		return "Environment"
	default:
		return string(id)
	}
}

func sectionBody(id sectionID, r SummaryReport) string {
	switch id {
	case secResult:
		return r.ExecutionResult
	case secOverview:
		return r.ExecutionOverview
	case secInfra:
		return r.Infrastructure
	case secValidation:
		return r.ValidationResults
	case secStages:
		return r.PipelineStages
	case secOutcome:
		return r.PipelineOutcome
	case secLearnings:
		return r.KeyLearnings
	case secRecommendations:
		return r.Recommendations
	case secSuccessStages:
		return r.SuccessfulStages
	case secFailed:
		return r.FailedStage
	case secSkipped:
		return r.SkippedStages
	case secInfraState:
		return r.InfrastructureState
	case secRecovery:
		return r.RecoverySteps
	case secProjectDetect:
		return r.ProjectDetection
	case secGeneratedFiles:
		return r.GeneratedFiles
	case secNextSteps:
		return r.NextSteps
	case secTunnelOverview:
		return r.TunnelOverview
	case secTunnelMetrics:
		return r.TunnelMetrics
	case secSessionOutcome:
		return r.SessionOutcome
	case secCleanup:
		return r.CleanupOverview
	case secResourcesRemoved:
		return r.ResourcesRemoved
	case secCluster:
		return r.ClusterStatus
	case secRegistry:
		return r.RegistryStatus
	case secJenkins:
		return r.JenkinsStatus
	case secEnvironment:
		return r.EnvironmentState
	default:
		return ""
	}
}
