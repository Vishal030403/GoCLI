package ai

import (
	"fmt"
	"strings"
)

// AnalysisResult holds a structured diagnosis from local rules or Gemini.
type AnalysisResult struct {
	Issue       string
	RootCause   string
	Explanation string
	Impact      string
	Resolution  []string
	Prevention  string
	FixCommands []string
	Source      string // "local" or "gemini"
}

// Recommendation holds a non-fatal advisory for warnings.
type Recommendation struct {
	Issue        string
	WhyItMatters string
	SuggestedFix string
	Benefits     []string
}

// AnalyzeFailure diagnoses a critical failure. Local rules run first; Gemini is a fallback.
func AnalyzeFailure(ctx FailureContext) AnalysisResult {
	if result, ok := matchKnownFailure(ctx); ok {
		return result
	}
	result, err := analyzeWithGemini(ctx)
	if err != nil {
		return fallbackResult(ctx, err)
	}
	return result
}

// AnalyzeLogs is the entry point for standalone log analysis (pipeline logs analyze).
// It parses logs into structured context before attempting local or Gemini diagnosis.
func AnalyzeLogs(logContent string) (AnalysisResult, error) {
	ctx := BuildFailureContext("pipeline logs analyze", "Log Analysis", "", 1, logContent)
	ctx.Command = "pipeline logs analyze"
	result := AnalyzeFailure(ctx)
	if result.Issue == "" && result.RootCause == "" {
		return result, fmt.Errorf("could not determine root cause")
	}
	return result, nil
}

func analyzeWithGemini(ctx FailureContext) (AnalysisResult, error) {
	var result AnalysisResult

	client, err := NewClient()
	if err != nil {
		return result, err
	}
	defer client.Close()

	systemPrompt := `You are a Senior DevOps Engineer.
Analyze this failure and respond with ONLY valid JSON in this exact structure:
{
  "issue": "string — short title of the problem",
  "root_cause": "string",
  "explanation": "string — why it happened, for a developer with limited DevOps knowledge",
  "impact": "string",
  "resolution": ["string step 1", "string step 2"],
  "prevention": "string",
  "fix_commands": ["optional shell command"]
}
Keep each field concise and actionable.`

	logBlock := strings.Join(ctx.RecentLogs, "\n")
	if logBlock == "" {
		logBlock = "(no recent log lines captured)"
	}

	skipped := "none"
	if len(ctx.SkippedStages) > 0 {
		skipped = strings.Join(ctx.SkippedStages, ", ")
	}

	userMessage := fmt.Sprintf(`Analyze this failure.

Command:
%s

Failed Stage:
%s

Error:
%s

Exit Code:
%d

Skipped Stages:
%s

Recent Logs:
%s`,
		ctx.Command, ctx.FailedStage, ctx.Error, ctx.ExitCode, skipped, logBlock)

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
	}
	var raw rawResult
	if err := parseJSON(cleaned, &raw); err != nil {
		return result, fmt.Errorf("AI returned invalid JSON: %w", err)
	}

	result = AnalysisResult{
		Issue:       raw.Issue,
		RootCause:   raw.RootCause,
		Explanation: raw.Explanation,
		Impact:      raw.Impact,
		Resolution:  raw.Resolution,
		Prevention:  raw.Prevention,
		FixCommands: raw.FixCommands,
		Source:      "gemini",
	}
	return result, nil
}

func fallbackResult(ctx FailureContext, err error) AnalysisResult {
	return AnalysisResult{
		Issue:       ctx.Stage + " failed",
		RootCause:   ctx.Error,
		Explanation: "Automated analysis could not reach Gemini. Review the error and recent logs below.",
		Impact:      "The pipeline stopped at this stage.",
		Resolution: []string{
			"Review the error output above.",
			fmt.Sprintf("Re-run after fixing: %s", ctx.Stage),
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

// HandleFailure runs analysis and prints results for a failed stage. Does not exit.
func HandleFailure(ctx FailureContext) {
	result := AnalyzeFailure(ctx)
	PrintAnalysis(result)
}
