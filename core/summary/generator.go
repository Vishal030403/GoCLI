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
			ch <- sanitizeReport(state, compactForTerminal(r))
			return
		}
		ch <- SummaryReport{}
	}()

	select {
	case r := <-ch:
		if r.hasContent() {
			r.ExecutionResult = buildExecutionResult(state)
			if r.PipelineStages == "" {
				r.PipelineStages = formatStagesBrief(state)
			}
			r.RawMarkdown = buildMarkdown(state, r)
			return r
		}
	case <-time.After(20 * time.Second):
	}

	report = sanitizeReport(state, compactForTerminal(report))
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

	userMsg := fmt.Sprintf("Brief summary. JSON data only:\n%s", string(payload))

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

	return rawToReport(raw), nil
}

func rawToReport(raw geminiSummaryRaw) SummaryReport {
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
	}
}

func generateFallback(state ExecutionState) SummaryReport {
	stages := formatStagesBrief(state)
	r := SummaryReport{
		ExecutionResult:  buildExecutionResult(state),
		PipelineStages:   stages,
		ProjectDetection: state.Metadata["framework"],
	}

	switch commandShort(state.Command) {
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
		ok := SuccessfulStageNames(state)
		if len(ok) > 0 {
			r.SuccessfulStages = strings.Join(ok, ", ")
		}
		skip := SkippedStageNames(state)
		if len(skip) > 0 {
			r.SkippedStages = strings.Join(skip, ", ")
		}
		if state.FailedStage != "" {
			r.FailedStage = displayStageName(state.FailedStage)
		}
		r.RecoverySteps = fallbackRecovery(state)
	}

	return r
}

func buildInitFallback(state ExecutionState, r SummaryReport) SummaryReport {
	fw := state.Metadata["framework"]
	r.ExecutionOverview = fmt.Sprintf("Scaffolding ready (%s).", fw)
	r.GeneratedFiles = "Dockerfile, Jenkinsfile, k8s/, pipeline.yaml"
	r.NextSteps = "Run: pipeline prep-ci"
	r.PipelineOutcome = "Init complete"
	return r
}

func buildPrepCIFallback(state ExecutionState, r SummaryReport) SummaryReport {
	r.ExecutionOverview = "Sandbox ready."
	r.Infrastructure = formatInfrastructureBrief(state)
	r.ValidationResults = "Docker, Kind, kubectl, ports OK"
	r.PipelineOutcome = "Jenkins + registry + Kind cluster up"
	r.Recommendations = "Jenkins :8080 → pipeline tunnel"
	return r
}

func buildTunnelFallback(state ExecutionState, r SummaryReport) SummaryReport {
	r.TunnelMetrics = formatTunnelMetrics(state)
	r.TunnelOverview = r.TunnelMetrics
	r.SessionOutcome = r.TunnelMetrics
	r.PipelineOutcome = "Tunnel closed"
	r.Recommendations = "http://localhost:8081 when tunnel is open"
	return r
}

func buildDestroyCIFallback(state ExecutionState, r SummaryReport) SummaryReport {
	r.CleanupOverview = "Sandbox torn down."
	r.ResourcesRemoved = formatInfrastructureBrief(state)
	r.EnvironmentState = "Ports freed, kubeconfig restored"
	r.Recommendations = "pipeline prep-ci for a fresh sandbox"
	return r
}

func buildGenericFallback(state ExecutionState, r SummaryReport) SummaryReport {
	r.ExecutionOverview = commandShort(state.Command) + " done"
	r.PipelineOutcome = "Complete"
	return r
}

func fallbackRecovery(state ExecutionState) string {
	return fmt.Sprintf("Fix %s, then: pipeline %s",
		displayStageName(state.FailedStage), commandShort(state.Command))
}

func (r SummaryReport) hasContent() bool {
	return r.ExecutionOverview != "" || r.TunnelMetrics != "" || r.CleanupOverview != ""
}
