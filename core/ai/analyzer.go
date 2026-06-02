package ai

import (
	"fmt"
	"strings"
)

// AnalysisResult holds a structured diagnosis from local rules or Gemini.
type AnalysisResult struct {
	Issue          string
	RootCause      string
	Explanation    string
	Impact         string
	Resolution     []string
	Prevention     string
	FixCommands    []string
	Confidence     Confidence
	Evidence       []string
	PossibleCauses []string
	Source         string // "local" or "gemini"
}

// Recommendation holds a non-fatal advisory for warnings.
type Recommendation struct {
	Issue        string
	WhyItMatters string
	SuggestedFix string
	Benefits     []string
}

// AnalyzeFailure diagnoses a critical failure using evidence collection, classification, then Gemini.
func AnalyzeFailure(ctx DiagnosticContext) AnalysisResult {
	CollectEvidence(&ctx)

	if result, conf, ok := Classify(ctx); ok {
		result.Confidence = conf
		result.Evidence = ctx.Evidence
		return result
	}

	result, err := analyzeWithGemini(ctx)
	if err != nil {
		fb := fallbackResult(ctx, err)
		fb.Evidence = ctx.Evidence
		fb.Confidence = ConfidenceLow
		fb.PossibleCauses = lowConfidenceHypotheses(ctx)
		return fb
	}
	result.Confidence = inferGeminiConfidence(ctx, result)
	result.Evidence = ctx.Evidence
	if result.Confidence == ConfidenceLow {
		result.PossibleCauses = lowConfidenceHypotheses(ctx)
	}
	return result
}

// AnalyzeLogs is the entry point for standalone log analysis (pipeline logs analyze).
func AnalyzeLogs(logContent string) (AnalysisResult, error) {
	ctx := BuildFailureContext("pipeline logs analyze", "Log Analysis", "", 1, logContent)
	ctx.Command = "pipeline logs analyze"
	result := AnalyzeFailure(ctx)
	if result.Confidence == ConfidenceLow && result.Issue == "" && result.RootCause == "" {
		return result, fmt.Errorf("could not determine root cause")
	}
	if result.Issue == "" && result.RootCause == "" && len(result.PossibleCauses) == 0 {
		return result, fmt.Errorf("could not determine root cause")
	}
	return result, nil
}

func analyzeWithGemini(ctx DiagnosticContext) (AnalysisResult, error) {
	var result AnalysisResult

	client, err := NewClient()
	if err != nil {
		return result, err
	}
	defer client.Close()

	systemPrompt := `You are a Senior DevOps Engineer.
Diagnose based ONLY on the verified evidence provided. Do not assume facts not present in the evidence.
If evidence is insufficient, say so in the explanation and list plausible causes without claiming certainty.

Respond with ONLY valid JSON in this exact structure:
{
  "issue": "string — short title",
  "root_cause": "string",
  "explanation": "string — for a developer with limited DevOps knowledge",
  "impact": "string",
  "resolution": ["step 1", "step 2"],
  "prevention": "string",
  "fix_commands": ["optional shell command"],
  "confidence": "High" or "Medium" or "Low"
}
Keep each field concise and actionable.`

	userMessage := buildGeminiPayload(ctx)

	responseText, err := client.Complete(systemPrompt, userMessage)
	if err != nil {
		return result, fmt.Errorf("gemini analysis failed: %w", err)
	}

	cleaned := cleanJSONResponse(responseText)

	type rawResult struct {
		Issue       string   `json:"issue"`
		RootCause   string   `json:"root_cause"`
		Explanation string   `json:"explanation"`
		Impact      string   `json:"impact"`
		Resolution  []string `json:"resolution"`
		Prevention  string   `json:"prevention"`
		FixCommands []string `json:"fix_commands"`
		Confidence  string   `json:"confidence"`
	}
	var raw rawResult
	if err := parseJSON(cleaned, &raw); err != nil {
		return result, fmt.Errorf("AI returned invalid JSON: %w", err)
	}

	conf := ConfidenceMedium
	switch strings.ToLower(raw.Confidence) {
	case "high":
		conf = ConfidenceHigh
	case "low":
		conf = ConfidenceLow
	}

	result = AnalysisResult{
		Issue:       raw.Issue,
		RootCause:   raw.RootCause,
		Explanation: raw.Explanation,
		Impact:      raw.Impact,
		Resolution:  raw.Resolution,
		Prevention:  raw.Prevention,
		FixCommands: raw.FixCommands,
		Confidence:  conf,
		Source:      "gemini",
	}
	return result, nil
}

// buildGeminiPayload formats structured evidence for Gemini (no full log dumps).
func buildGeminiPayload(ctx DiagnosticContext) string {
	var b strings.Builder
	b.WriteString("Analyze this failure using ONLY the verified evidence below.\n\n")

	b.WriteString("Command:\n")
	b.WriteString(ctx.Command + "\n\n")

	b.WriteString("Failed Stage:\n")
	stage := ctx.FailedStage
	if stage == "" {
		stage = ctx.Stage
	}
	b.WriteString(stage + "\n\n")

	b.WriteString("Error:\n")
	b.WriteString(ctx.Error + "\n\n")

	fmt.Fprintf(&b, "Exit Code: %d\n\n", ctx.ExitCode)

	if ctx.FinalStatus != "" {
		b.WriteString("Pipeline Result: " + ctx.FinalStatus + "\n\n")
	}
	if len(ctx.SkippedStages) > 0 {
		b.WriteString("Skipped Stages: " + strings.Join(ctx.SkippedStages, ", ") + "\n\n")
	}

	b.WriteString("Verified Evidence:\n")
	writeEvidenceLine(&b, "Pod Status", ctx.PodStatus)
	if ctx.PodName != "" {
		writeEvidenceLine(&b, "Pod", ctx.PodNamespace+"/"+ctx.PodName)
	}
	if ctx.PodRestarts > 0 {
		fmt.Fprintf(&b, "* Pod Restarts: %d\n", ctx.PodRestarts)
	}
	writeEvidenceLine(&b, "Service Exists", boolStr(ctx.ServiceExists))
	writeEvidenceLine(&b, "Registry Reachable", boolStr(ctx.RegistryReachable))
	writeEvidenceLine(&b, "Registry Status", ctx.RegistryStatus)
	writeEvidenceLine(&b, "Docker Daemon OK", boolStr(ctx.DockerDaemonOK))
	writeEvidenceLine(&b, "Jenkins Status", ctx.JenkinsStatus)
	if ctx.PortForwardFailed {
		b.WriteString("* Port Forward Failed: Yes\n")
	}
	if ctx.TunnelEstablished {
		b.WriteString("* Tunnel Established: Yes\n")
	}
	for _, e := range ctx.Evidence {
		b.WriteString("* " + e + "\n")
	}

	// Trimmed log summary only (not full pipeline output)
	if summary := logSummary(ctx.RecentLogs); summary != "" {
		b.WriteString("\nLog Summary (trimmed):\n")
		b.WriteString(summary)
	}

	b.WriteString("\nProvide: root cause, explanation, impact, resolution, prevention.\n")
	return b.String()
}

func writeEvidenceLine(b *strings.Builder, label, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "* %s: %s\n", label, value)
}

func boolStr(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func logSummary(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	max := 8
	if len(lines) > max {
		lines = lines[len(lines)-max:]
	}
	return strings.Join(lines, "\n")
}

func inferGeminiConfidence(ctx DiagnosticContext, result AnalysisResult) Confidence {
	if result.Confidence != "" {
		return result.Confidence
	}
	if hasStrongEvidence(ctx) && result.RootCause != "" {
		return ConfidenceMedium
	}
	return ConfidenceLow
}

func fallbackResult(ctx DiagnosticContext, err error) AnalysisResult {
	stage := ctx.Stage
	if ctx.FailedStage != "" {
		stage = ctx.FailedStage
	}
	return AnalysisResult{
		Issue:       stage + " failed",
		RootCause:   ctx.Error,
		Explanation: "Automated analysis could not reach Gemini. Review the error and verified evidence below.",
		Impact:      "The pipeline stopped at this stage.",
		Resolution: []string{
			"Review the error output above.",
			fmt.Sprintf("Re-run after fixing: %s", stage),
		},
		Prevention: fmt.Sprintf("Analysis unavailable: %v", err),
		Source:     "local",
	}
}

func cleanJSONResponse(s string) string {
	cleaned := strings.TrimSpace(s)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	return strings.TrimSpace(cleaned)
}

// RecommendForWarnings produces local recommendations for policy warnings (no Gemini).
func RecommendForWarnings(warnings []WarningItem) []Recommendation {
	if len(warnings) == 0 {
		return nil
	}
	var recs []Recommendation
	for _, w := range warnings {
		if rec, ok := warningRecommendation(w); ok {
			recs = append(recs, rec)
		}
	}
	return recs
}

func warningRecommendation(w WarningItem) (Recommendation, bool) {
	switch w.PolicyName {
	case "health-endpoint":
		return Recommendation{
			Issue:        "Health endpoint missing",
			WhyItMatters: "Health endpoints allow Kubernetes to determine whether an application is healthy.",
			SuggestedFix: "Add a GET /health (or /healthz) route that returns HTTP 200 when the app is ready.",
			Benefits: []string{
				"Better monitoring and alerting",
				"Faster recovery during rollouts",
				"Improved reliability under load",
			},
		}, true
	case "resource-limits":
		return Recommendation{
			Issue:        "Missing resource limits",
			WhyItMatters: "Without CPU/memory limits, a pod can starve other workloads or be evicted unpredictably.",
			SuggestedFix: "Add resources.requests and resources.limits to each container in your Deployment manifests.",
			Benefits: []string{
				"Predictable scheduling",
				"Cost control in shared clusters",
				"Fewer OOMKill surprises",
			},
		}, true
	case "mandatory-probes":
		return Recommendation{
			Issue:        "Missing readiness/liveness probes",
			WhyItMatters: "Probes tell Kubernetes when your app is ready for traffic and when to restart unhealthy pods.",
			SuggestedFix: "Add readinessProbe and livenessProbe pointing at your health endpoint.",
			Benefits: []string{
				"Zero-downtime deployments",
				"Automatic recovery from hung processes",
			},
		}, true
	case "logging-standard":
		return Recommendation{
			Issue:        "Non-standard logging detected",
			WhyItMatters: "Structured logs improve searchability in centralized logging systems.",
			SuggestedFix: "Use structured JSON logging or your platform's standard logger instead of fmt.Print/debug prints.",
			Benefits: []string{
				"Easier log aggregation",
				"Better incident debugging",
			},
		}, true
	case "env-var-size":
		return Recommendation{
			Issue:        "Large environment variable values",
			WhyItMatters: "Very large env vars can hit Kubernetes limits and expose secrets in process listings.",
			SuggestedFix: "Move large config to ConfigMaps/Secrets and mount or reference them instead.",
			Benefits: []string{
				"Safer secret handling",
				"Cleaner pod specs",
			},
		}, true
	case "api-versioning":
		return Recommendation{
			Issue:        "API versioning not detected",
			WhyItMatters: "Versioned APIs let you evolve endpoints without breaking existing clients.",
			SuggestedFix: "Prefix routes with /api/v1 (or use header-based versioning).",
			Benefits: []string{
				"Safer API evolution",
				"Clear contract for consumers",
			},
		}, true
	default:
		if w.Message != "" {
			return Recommendation{
				Issue:        w.PolicyName,
				WhyItMatters: "This policy check flagged a potential reliability or security gap.",
				SuggestedFix: w.Message,
				Benefits:     []string{"Improved platform compliance"},
			}, true
		}
	}
	return Recommendation{}, false
}

// PrintAnalysis prints critical failure analysis in the standard format.
func PrintAnalysis(result AnalysisResult) {
	fmt.Println()
	fmt.Println("\033[1;36m🤖 AI Analysis\033[0m")
	fmt.Println()

	if result.Issue != "" {
		fmt.Println("\033[1mIssue:\033[0m")
		fmt.Printf("   %s\n", result.Issue)
		fmt.Println()
	}

	if len(result.Evidence) > 0 {
		fmt.Println("\033[1mEvidence:\033[0m")
		for _, e := range result.Evidence {
			fmt.Printf("   • %s\n", e)
		}
		fmt.Println()
	}

	if result.Confidence != "" {
		fmt.Println("\033[1mConfidence:\033[0m")
		fmt.Printf("   %s\n", result.Confidence)
		fmt.Println()
	}

	if result.RootCause != "" {
		fmt.Println("\033[1mRoot Cause:\033[0m")
		fmt.Printf("   %s\n", result.RootCause)
		fmt.Println()
	}

	if result.Explanation != "" {
		fmt.Println("\033[1mExplanation:\033[0m")
		fmt.Printf("   %s\n", result.Explanation)
		fmt.Println()
	}

	if result.Impact != "" {
		fmt.Println("\033[1mImpact:\033[0m")
		fmt.Printf("   %s\n", result.Impact)
		fmt.Println()
	}

	if len(result.PossibleCauses) > 0 && result.Confidence == ConfidenceLow {
		fmt.Println("\033[1mPossible Causes:\033[0m")
		for i, c := range result.PossibleCauses {
			fmt.Printf("   %d. %s\n", i+1, c)
		}
		fmt.Println()
	}

	if len(result.Resolution) > 0 {
		fmt.Println("\033[1mResolution:\033[0m")
		for i, step := range result.Resolution {
			fmt.Printf("   %d. %s\n", i+1, step)
		}
		fmt.Println()
	}

	if result.Prevention != "" {
		fmt.Println("\033[1mPrevention:\033[0m")
		fmt.Printf("   %s\n", result.Prevention)
		fmt.Println()
	}

	if len(result.FixCommands) > 0 {
		for _, cmd := range result.FixCommands {
			fmt.Printf("   \033[0;32m$ %s\033[0m\n", cmd)
		}
		fmt.Println()
	}
}

// PrintRecommendations prints warning-case advisory output.
func PrintRecommendations(recs []Recommendation) {
	if len(recs) == 0 {
		return
	}
	for _, rec := range recs {
		fmt.Println()
		fmt.Printf("\033[1;33m⚠ Warning: %s\033[0m\n", rec.Issue)
		fmt.Println()
		fmt.Println("\033[1;36m🤖 AI Recommendation\033[0m")
		fmt.Println()
		if rec.WhyItMatters != "" {
			fmt.Println("\033[1mWhy this matters:\033[0m")
			fmt.Printf("   %s\n", rec.WhyItMatters)
			fmt.Println()
		}
		if rec.SuggestedFix != "" {
			fmt.Println("\033[1mSuggested Fix:\033[0m")
			fmt.Printf("   %s\n", rec.SuggestedFix)
			fmt.Println()
		}
		if len(rec.Benefits) > 0 {
			fmt.Println("\033[1mBenefits:\033[0m")
			for _, b := range rec.Benefits {
				fmt.Printf("   • %s\n", b)
			}
			fmt.Println()
		}
	}
}

// HandleFailure collects evidence, runs analysis, and prints results. Does not exit.
func HandleFailure(ctx FailureContext) {
	result := AnalyzeFailure(ctx)
	PrintAnalysis(result)
}
