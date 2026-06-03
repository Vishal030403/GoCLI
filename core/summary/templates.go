package summary

// sectionID identifies a summary block.
type sectionID string

const (
	secResult           sectionID = "execution_result"
	secOverview         sectionID = "execution_overview"
	secInfra            sectionID = "infrastructure"
	secValidation       sectionID = "validation"
	secStages           sectionID = "pipeline_stages"
	secOutcome          sectionID = "pipeline_outcome"
	secLearnings        sectionID = "key_learnings"
	secRecommendations  sectionID = "recommendations"
	secSuccessStages    sectionID = "successful_stages"
	secFailed           sectionID = "failed_stage"
	secSkipped          sectionID = "skipped_stages"
	secInfraState       sectionID = "infrastructure_state"
	secRecovery         sectionID = "recovery"
	secProjectDetect    sectionID = "project_detection"
	secGeneratedFiles   sectionID = "generated_files"
	secNextSteps        sectionID = "next_steps"
	secTunnelOverview   sectionID = "tunnel_overview"
	secTunnelMetrics    sectionID = "tunnel_metrics"
	secSessionOutcome   sectionID = "session_outcome"
	secCleanup          sectionID = "cleanup_overview"
	secResourcesRemoved sectionID = "resources_removed"
	secCluster          sectionID = "cluster_status"
	secRegistry         sectionID = "registry_status"
	secJenkins          sectionID = "jenkins_status"
	secEnvironment      sectionID = "environment_state"
)

type templateLayout struct {
	terminalSuccess []sectionID
	terminalFailure []sectionID
	markdownExtra   []sectionID
}

var commandTemplates = map[string]templateLayout{
	"init": {
		terminalSuccess: []sectionID{secResult, secOverview, secValidation, secProjectDetect, secGeneratedFiles, secNextSteps},
		terminalFailure: []sectionID{secResult, secSuccessStages, secFailed, secSkipped, secRecovery, secRecommendations},
		markdownExtra:   []sectionID{secStages, secLearnings, secOutcome},
	},
	"prep-ci": {
		terminalSuccess: []sectionID{secResult, secOverview, secInfra, secOutcome, secLearnings, secRecommendations},
		terminalFailure: []sectionID{secResult, secSuccessStages, secFailed, secSkipped, secInfraState, secRecovery, secRecommendations},
		markdownExtra:   []sectionID{secValidation, secStages, secOutcome},
	},
	"tunnel": {
		terminalSuccess: []sectionID{secResult, secTunnelOverview, secTunnelMetrics, secSessionOutcome, secLearnings, secRecommendations},
		terminalFailure: []sectionID{secResult, secTunnelOverview, secTunnelMetrics, secFailed, secRecovery, secRecommendations},
		markdownExtra:   []sectionID{secOverview, secStages},
	},
	"destroy-ci": {
		terminalSuccess: []sectionID{secResult, secCleanup, secResourcesRemoved, secEnvironment, secRecommendations},
		terminalFailure: []sectionID{secResult, secCleanup, secFailed, secRecovery, secRecommendations},
		markdownExtra:   []sectionID{secCluster, secRegistry, secJenkins, secStages, secLearnings},
	},
}

func layoutFor(state ExecutionState) templateLayout {
	cmd := commandShort(state.Command)
	if t, ok := commandTemplates[cmd]; ok {
		return t
	}
	return templateLayout{
		terminalSuccess: []sectionID{secResult, secOverview, secInfra, secOutcome, secLearnings, secRecommendations},
		terminalFailure: []sectionID{secResult, secSuccessStages, secFailed, secSkipped, secRecovery, secRecommendations},
		markdownExtra:   []sectionID{secValidation, secStages},
	}
}

func sectionTitle(id sectionID) string {
	switch id {
	case secResult:
		return "Execution Result"
	case secOverview:
		return "Execution Overview"
	case secInfra:
		return "Infrastructure Created"
	case secValidation:
		return "Validation Results"
	case secStages:
		return "Pipeline Flow"
	case secOutcome:
		return "Pipeline Outcome"
	case secLearnings:
		return "Key Learnings"
	case secRecommendations:
		return "Recommendations"
	case secSuccessStages:
		return "Successful Stages"
	case secFailed:
		return "Failed Stage"
	case secSkipped:
		return "Skipped Stages"
	case secInfraState:
		return "Infrastructure State"
	case secRecovery:
		return "Recovery Recommendations"
	case secProjectDetect:
		return "Project Detection"
	case secGeneratedFiles:
		return "Generated Files"
	case secNextSteps:
		return "Developer Next Steps"
	case secTunnelOverview:
		return "Tunnel Overview"
	case secTunnelMetrics:
		return "Tunnel Session"
	case secSessionOutcome:
		return "Session Outcome"
	case secCleanup:
		return "Cleanup Overview"
	case secResourcesRemoved:
		return "Resources Removed"
	case secCluster:
		return "Cluster Status"
	case secRegistry:
		return "Registry Status"
	case secJenkins:
		return "Jenkins Status"
	case secEnvironment:
		return "Environment State"
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
