package summary

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pipeline-cli/core/ai"
)

type geminiSummaryRaw struct {
	ExecutionOverview   string `json:"execution_overview"`
	Infrastructure      string `json:"infrastructure_created"`
	ValidationResults   string `json:"validation_results"`
	PipelineStages      string `json:"pipeline_stages"`
	PipelineOutcome     string `json:"pipeline_outcome"`
	KeyLearnings        string `json:"key_learnings"`
	Recommendations     string `json:"recommendations"`
	SuccessfulStages    string `json:"successful_stages"`
	FailedStage         string `json:"failed_stage"`
	SkippedStages       string `json:"skipped_stages"`
	InfrastructureState string `json:"infrastructure_state"`
	RecoverySteps       string `json:"recovery_steps"`
	ProjectDetection    string `json:"project_detection"`
	GeneratedFiles      string `json:"generated_files"`
	NextSteps           string `json:"next_steps"`
	TunnelOverview      string `json:"tunnel_overview"`
	TunnelMetrics       string `json:"tunnel_metrics"`
	SessionOutcome      string `json:"session_outcome"`
	CleanupOverview     string `json:"cleanup_overview"`
	ResourcesRemoved    string `json:"resources_removed"`
	ClusterStatus       string `json:"cluster_status"`
	RegistryStatus      string `json:"registry_status"`
	JenkinsStatus       string `json:"jenkins_status"`
	EnvironmentState    string `json:"environment_state"`
	DeveloperNotes      string `json:"developer_notes"`
}

func generateReport(state ExecutionState) SummaryReport {
	report := generateFallback(state)

	ch := make(chan SummaryReport, 1)
	go func() {
		r, err := generateWithGemini(state)
		if err == nil && r.hasContent() {
			ch <- sanitizeReport(state, r)
			return
		}
		ch <- SummaryReport{}
	}()

	select {
	case r := <-ch:
		if r.hasContent() {
			r.ExecutionResult = buildExecutionResult(state)
			if r.PipelineStages == "" {
				r.PipelineStages = formatStagesConcise(state)
			}
			r.RawMarkdown = buildMarkdown(state, r)
			return r
		}
	case <-time.After(25 * time.Second):
	}

	report = sanitizeReport(state, report)
	report.RawMarkdown = buildMarkdown(state, report)
	return report
}

func generateWithGemini(state ExecutionState) (SummaryReport, error) {
	var empty SummaryReport

	client, err := ai.NewClient()
	if err != nil {
		return empty, err
	}
	defer client.Close()

	payload, err := json.Marshal(StructuredPayload(state))
	if err != nil {
		return empty, err
	}

	userMsg := fmt.Sprintf("Generate a command-aware summary from this structured execution data ONLY:\n\n%s", string(payload))

	text, err := client.Complete(geminiSystemPrompt, userMsg)
	if err != nil {
		return empty, err
	}

	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var raw geminiSummaryRaw
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return empty, err
	}

	return SummaryReport{
		ExecutionOverview:   raw.ExecutionOverview,
		Infrastructure:      raw.Infrastructure,
		ValidationResults:   raw.ValidationResults,
		PipelineStages:      raw.PipelineStages,
		PipelineOutcome:     raw.PipelineOutcome,
		KeyLearnings:        raw.KeyLearnings,
		Recommendations:     raw.Recommendations,
		SuccessfulStages:    raw.SuccessfulStages,
		FailedStage:         raw.FailedStage,
		SkippedStages:       raw.SkippedStages,
		InfrastructureState: raw.InfrastructureState,
		RecoverySteps:       raw.RecoverySteps,
		ProjectDetection:    raw.ProjectDetection,
		GeneratedFiles:      raw.GeneratedFiles,
		NextSteps:           raw.NextSteps,
		TunnelOverview:      raw.TunnelOverview,
		TunnelMetrics:       raw.TunnelMetrics,
		SessionOutcome:      raw.SessionOutcome,
		CleanupOverview:     raw.CleanupOverview,
		ResourcesRemoved:    raw.ResourcesRemoved,
		ClusterStatus:       raw.ClusterStatus,
		RegistryStatus:      raw.RegistryStatus,
		JenkinsStatus:       raw.JenkinsStatus,
		EnvironmentState:    raw.EnvironmentState,
		DeveloperNotes:      raw.DeveloperNotes,
	}, nil
}

func generateFallback(state ExecutionState) SummaryReport {
	cmd := commandShort(state.Command)
	stages := formatStagesConcise(state)
	result := buildExecutionResult(state)

	r := SummaryReport{
		ExecutionResult:  result,
		PipelineStages:   stages,
		ProjectDetection: state.Metadata["framework"],
	}

	switch cmd {
	case "init":
		r = buildInitFallback(state, r)
	case "prep-ci":
		r = buildPrepCIFallback(state, r)
	case "tunnel":
		r = buildTunnelFallback(state, r)
	case "destroy-ci":
		r = buildDestroyCIFallback(state, r)
	default:
		r = buildGenericFallback(state, r)
	}

	if !state.Success {
		r.SuccessfulStages = strings.Join(SuccessfulStageNames(state), ", ")
		r.SkippedStages = strings.Join(SkippedStageNames(state), ", ")
		if state.FailedStage != "" {
			r.FailedStage = displayStageName(state.FailedStage)
		}
		r.RecoverySteps = fallbackRecovery(state)
		r.InfrastructureState = r.Infrastructure
	}

	return r
}

func buildInitFallback(state ExecutionState, r SummaryReport) SummaryReport {
	r.ExecutionOverview = fmt.Sprintf("Initialized project scaffolding for **%s**.", state.Metadata["framework"])
	r.ValidationResults = "Framework detection and file generation completed."
	r.GeneratedFiles = "Dockerfile, Jenkinsfile, k8s manifests, pipeline.yaml (if new)"
	r.NextSteps = "Run `pipeline prep-ci` to provision local CI/CD, then push a build through Jenkins."
	r.KeyLearnings = fallbackLearnings(state)
	r.Recommendations = fallbackRecommendations(state)
	r.PipelineOutcome = "Scaffolding ready for local pipeline."
	return r
}

func buildPrepCIFallback(state ExecutionState, r SummaryReport) SummaryReport {
	r.ExecutionOverview = "Local CI/CD sandbox is provisioned and ready for builds."
	r.Infrastructure = formatInfrastructure(state)
	r.ValidationResults = "Preflight checks passed: Docker, Kind, kubectl, ports 5001/8080."
	r.PipelineOutcome = "Sandbox live — Jenkins, registry, and Kind cluster operational."
	r.KeyLearnings = fallbackLearnings(state)
	r.Recommendations = fallbackRecommendations(state)
	return r
}

func buildTunnelFallback(state ExecutionState, r SummaryReport) SummaryReport {
	if state.Tunnel != nil {
		t := state.Tunnel
		r.TunnelOverview = fmt.Sprintf("Port-forward from **localhost:%s** to service **%s** in namespace **%s**.",
			t.LocalPort, t.AppName, t.Namespace)
		r.TunnelMetrics = formatTunnelMetrics(state)
		r.SessionOutcome = t.Outcome
		if r.SessionOutcome == "" {
			r.SessionOutcome = "Session ended"
		}
	}
	r.KeyLearnings = fallbackLearnings(state)
	r.Recommendations = fallbackRecommendations(state)
	r.PipelineOutcome = r.SessionOutcome
	return r
}

func buildDestroyCIFallback(state ExecutionState, r SummaryReport) SummaryReport {
	r.CleanupOverview = "Tore down local CI/CD sandbox resources."
	r.ResourcesRemoved = formatInfrastructure(state)
	r.ClusterStatus = "Kind cluster ephemeral-test removed"
	r.RegistryStatus = "local-registry container and volume removed"
	r.JenkinsStatus = "local-jenkins container and volume removed"
	r.EnvironmentState = "Ports 5001 and 8080 freed; kubeconfig restored"
	r.Recommendations = "Run `pipeline prep-ci` when you need a fresh sandbox."
	return r
}

func buildGenericFallback(state ExecutionState, r SummaryReport) SummaryReport {
	r.ExecutionOverview = fmt.Sprintf("Completed **%s**.", commandShort(state.Command))
	r.Infrastructure = formatInfrastructure(state)
	r.PipelineOutcome = "Done"
	r.KeyLearnings = fallbackLearnings(state)
	r.Recommendations = fallbackRecommendations(state)
	return r
}

func formatInfrastructure(state ExecutionState) string {
	if len(state.Infrastructure) == 0 {
		return ""
	}
	var b strings.Builder
	for _, item := range state.Infrastructure {
		fmt.Fprintf(&b, "• %s — %s\n", item.Name, item.Detail)
	}
	return strings.TrimSpace(b.String())
}

func fallbackLearnings(state ExecutionState) string {
	switch commandShort(state.Command) {
	case "prep-ci":
		return "A local Kind cluster plus registry and Jenkins mirrors a minimal cloud pipeline on your laptop."
	case "init":
		return "Scaffolding tells the pipeline how to build, test, and deploy your app without manual DevOps setup."
	case "tunnel":
		return "Port-forward exposes in-cluster services on localhost so you can test without public ingress."
	case "destroy-ci":
		return "Cleanup removes containers, volumes, and clusters so ports and disk space are freed."
	default:
		return "This CLI runs CI/CD locally with Docker and Kubernetes (Kind)."
	}
}

func fallbackRecommendations(state ExecutionState) string {
	if !state.Success {
		return "Resolve the failed stage, then re-run the command. Ensure Docker is running and ports 5001/8080 are free."
	}
	switch commandShort(state.Command) {
	case "prep-ci":
		return "Open Jenkins (http://localhost:8080), verify the pipeline build, then `pipeline tunnel` to reach the app."
	case "tunnel":
		return "Use http://localhost:8081 while the tunnel is active."
	case "init":
		return "Commit generated files, then run `pipeline prep-ci`."
	default:
		return "Keep pipeline.yaml in version control for repeatable builds."
	}
}

func fallbackRecovery(state ExecutionState) string {
	return fmt.Sprintf("1. Review AI Analysis above.\n2. Fix the issue at stage: %s.\n3. Re-run: pipeline %s",
		displayStageName(state.FailedStage), commandShort(state.Command))
}

func (r SummaryReport) hasContent() bool {
	return r.ExecutionOverview != "" || r.TunnelOverview != "" || r.CleanupOverview != ""
}
