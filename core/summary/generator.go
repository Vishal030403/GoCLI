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
	KeyLearnings        string `json:"key_learnings"`
	Recommendations     string `json:"recommendations"`
	SuccessfulStages    string `json:"successful_stages"`
	FailedStage         string `json:"failed_stage"`
	SkippedStages       string `json:"skipped_stages"`
	InfrastructureState string `json:"infrastructure_state"`
	RecoverySteps       string `json:"recovery_steps"`
	OverallStatus       string `json:"overall_status"`
}

func generateReport(state ExecutionState) SummaryReport {
	report, err := generateWithGemini(state)
	if err == nil && report.hasContent() {
		report.RawMarkdown = buildMarkdown(state, report)
		return report
	}
	fb := generateFallback(state)
	fb.RawMarkdown = buildMarkdown(state, fb)
	return fb
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

	userMsg := fmt.Sprintf(`Generate an educational execution summary from this structured data only:

%s`, string(payload))

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
		KeyLearnings:        raw.KeyLearnings,
		Recommendations:     raw.Recommendations,
		SuccessfulStages:    raw.SuccessfulStages,
		FailedStage:         raw.FailedStage,
		SkippedStages:       raw.SkippedStages,
		InfrastructureState: raw.InfrastructureState,
		RecoverySteps:       raw.RecoverySteps,
		OverallStatus:       raw.OverallStatus,
	}, nil
}

func generateFallback(state ExecutionState) SummaryReport {
	cmd := commandShort(state.Command)
	dur := "unknown"
	if state.Duration > 0 {
		dur = state.Duration.Round(time.Second).String()
	} else if !state.EndTime.IsZero() {
		dur = state.EndTime.Sub(state.StartTime).Round(time.Second).String()
	}

	var overview string
	if state.Success {
		overview = fmt.Sprintf("Command **%s** completed successfully in %s.", cmd, dur)
	} else {
		overview = fmt.Sprintf("Command **%s** did not complete successfully.", cmd)
		if state.FailedStage != "" {
			overview += fmt.Sprintf(" Failure occurred at stage **%s**.", state.FailedStage)
		}
	}

	var infra strings.Builder
	if len(state.Infrastructure) == 0 {
		infra.WriteString("No infrastructure items were recorded for this run.")
	} else {
		for _, item := range state.Infrastructure {
			fmt.Fprintf(&infra, "- **%s**: %s\n", item.Name, item.Detail)
		}
	}

	var validation strings.Builder
	for _, s := range state.Stages {
		if strings.Contains(strings.ToLower(s.Name), "preflight") ||
			strings.Contains(strings.ToLower(s.Name), "check") ||
			strings.Contains(strings.ToLower(s.Name), "validation") {
			fmt.Fprintf(&validation, "- %s: %s\n", s.Name, s.Status)
		}
	}
	if validation.Len() == 0 {
		validation.WriteString("Preflight and validation stages ran as part of the command lifecycle.")
	}

	var stages strings.Builder
	for _, s := range state.Stages {
		line := fmt.Sprintf("- %s: %s", s.Name, s.Status)
		if s.Message != "" && s.Status == StageFailed {
			line += fmt.Sprintf(" (%s)", truncate(s.Message, 80))
		}
		stages.WriteString(line + "\n")
	}

	learnings := fallbackLearnings(state)
	recs := fallbackRecommendations(state)

	successList := strings.Join(SuccessfulStageNames(state), ", ")
	skippedList := strings.Join(SkippedStageNames(state), ", ")

	var recovery, overall, infraState string
	if !state.Success {
		overall = "FAILED — review AI Analysis above, then apply recovery steps."
		if state.FailedStage != "" {
			overall += " Failed stage: " + state.FailedStage + "."
		}
		recovery = "1. Fix the error shown in AI Analysis.\n2. Re-run: pipeline " + cmd + "\n3. Verify Docker, Kind, and Jenkins are healthy if the failure was infrastructure-related."
		infraState = infra.String()
	} else {
		overall = "SUCCESS — sandbox and pipeline resources are ready for development."
	}

	return SummaryReport{
		ExecutionOverview:   overview,
		Infrastructure:      strings.TrimSpace(infra.String()),
		ValidationResults:   strings.TrimSpace(validation.String()),
		PipelineStages:      strings.TrimSpace(stages.String()),
		KeyLearnings:        learnings,
		Recommendations:     recs,
		SuccessfulStages:    successList,
		FailedStage:         state.FailedStage,
		SkippedStages:       skippedList,
		InfrastructureState: strings.TrimSpace(infraState),
		RecoverySteps:       strings.TrimSpace(recovery),
		OverallStatus:       overall,
	}
}

func fallbackLearnings(state ExecutionState) string {
	switch commandShort(state.Command) {
	case "prep-ci":
		return "prep-ci provisions a local Kind Kubernetes cluster, a Docker registry on port 5001, and Jenkins on port 8080. " +
			"Together they mimic a small cloud CI/CD sandbox on your machine without pushing to a remote cloud."
	case "init":
		return "init detects your application framework and generates Dockerfile, Jenkinsfile, and Kubernetes manifests " +
			"so the pipeline knows how to build, test, and deploy your project."
	case "tunnel":
		return "tunnel uses kubectl port-forward to expose your in-cluster Service on localhost (8081) " +
			"so you can hit the app from your browser without deploying a public ingress."
	case "destroy-ci":
		return "destroy-ci tears down local Jenkins, the registry, Kind cluster, and optional generated files " +
			"to free ports and disk space after experiments."
	default:
		return "This CLI runs CI/CD stages locally using Docker and Kubernetes (Kind) so you can learn DevOps workflows safely on your laptop."
	}
}

func fallbackRecommendations(state ExecutionState) string {
	if !state.Success {
		return "Address the failed stage before re-running. Check Docker Desktop is running and ports 5001/8080 are free."
	}
	switch commandShort(state.Command) {
	case "prep-ci":
		return "Open Jenkins at http://localhost:8080 (admin/admin), watch your pipeline build, then run pipeline tunnel to reach the app."
	case "tunnel":
		return "Visit http://localhost:8081 while the tunnel is open. Press Ctrl+C when finished."
	default:
		return "Commit pipeline.yaml and generated manifests to version control for repeatable builds."
	}
}

func buildMarkdown(state ExecutionState, r SummaryReport) string {
	var b strings.Builder
	b.WriteString("# AI Execution Summary\n\n")
	b.WriteString("**Command:** " + state.Command + "\n\n")
	if state.Duration > 0 {
		b.WriteString("**Duration:** " + state.Duration.Round(1e9).String() + "\n\n")
	}
	writeMDSection(&b, "Execution Overview", r.ExecutionOverview)
	if state.Success {
		writeMDSection(&b, "Infrastructure Created", r.Infrastructure)
		writeMDSection(&b, "Validation Results", r.ValidationResults)
	} else {
		writeMDSection(&b, "Successful Stages", r.SuccessfulStages)
		writeMDSection(&b, "Failed Stage", r.FailedStage)
		writeMDSection(&b, "Skipped Stages", r.SkippedStages)
		writeMDSection(&b, "Infrastructure State", r.InfrastructureState)
		writeMDSection(&b, "Recovery Steps", r.RecoverySteps)
	}
	writeMDSection(&b, "Pipeline Stages", r.PipelineStages)
	writeMDSection(&b, "Key Learnings", r.KeyLearnings)
	writeMDSection(&b, "Recommendations", r.Recommendations)
	if r.OverallStatus != "" {
		writeMDSection(&b, "Overall Status", r.OverallStatus)
	}
	return b.String()
}

func writeMDSection(b *strings.Builder, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	b.WriteString("## " + title + "\n\n")
	b.WriteString(body + "\n\n")
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (r SummaryReport) hasContent() bool {
	return r.ExecutionOverview != "" || r.PipelineStages != ""
}
