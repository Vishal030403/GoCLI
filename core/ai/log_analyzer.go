package ai

import (
	"fmt"
	"strings"
)

// AnalysisResult holds the structured diagnosis returned by the AI log analyzer.
type AnalysisResult struct {
	RootCause   string
	Suggestions []string
	FixCommands []string
}

// PrintAnalysis prints a formatted AI diagnosis to stdout.
func PrintAnalysis(result AnalysisResult) {
	fmt.Println()
	fmt.Println("\033[1;36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("\033[1;36m  🤖 AI Log Analysis\033[0m")
	fmt.Println("\033[1;36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")

	fmt.Println()
	fmt.Println("\033[1;31m🔍 Root Cause:\033[0m")
	fmt.Printf("   %s\n", result.RootCause)

	if len(result.Suggestions) > 0 {
		fmt.Println()
		fmt.Println("\033[1;33m💡 Suggestions:\033[0m")
		for i, s := range result.Suggestions {
			fmt.Printf("   %d. %s\n", i+1, s)
		}
	}

	if len(result.FixCommands) > 0 {
		fmt.Println()
		fmt.Println("\033[1;32m🔧 Fix Commands:\033[0m")
		for _, cmd := range result.FixCommands {
			fmt.Printf("   \033[0;32m$ %s\033[0m\n", cmd)
		}
	}

	fmt.Println()
	fmt.Println("\033[1;36m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
}

// AnalyzeLogs sends Jenkins or Kubernetes log output to Gemini and returns
// a structured diagnosis with actionable fix suggestions.
func AnalyzeLogs(logContent string) (AnalysisResult, error) {
	var result AnalysisResult

	client, err := NewClient()
	if err != nil {
		return result, err
	}
	defer client.Close()

	systemPrompt := `You are a DevOps engineer analyzing CI/CD pipeline logs.
Identify the root cause of the failure and provide specific, actionable fixes.
Respond with ONLY valid JSON in this exact structure:
{
  "root_cause": "string — one sentence describing the exact cause",
  "suggestions": ["string", "string"],
  "fix_commands": ["string — exact shell command to run", "string"]
}`

	truncated := logContent
	if len(truncated) > 15000 {
		truncated = "...[truncated]\n" + truncated[len(truncated)-15000:]
	}

	userMessage := fmt.Sprintf("Analyze this pipeline log and return diagnosis JSON:\n\n%s", truncated)

	responseText, err := client.Complete(systemPrompt, userMessage)
	if err != nil {
		return result, fmt.Errorf("log analysis failed: %w", err)
	}

	cleaned := strings.TrimSpace(responseText)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	type rawResult struct {
		RootCause   string   `json:"root_cause"`
		Suggestions []string `json:"suggestions"`
		FixCommands []string `json:"fix_commands"`
	}
	var raw rawResult

	if err := parseJSON(cleaned, &raw); err != nil {
		return result, fmt.Errorf("AI returned invalid JSON: %w", err)
	}

	result.RootCause = raw.RootCause
	result.Suggestions = raw.Suggestions
	result.FixCommands = raw.FixCommands

	return result, nil
}
