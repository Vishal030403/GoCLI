package summary

import "fmt"

// PrintSummary renders the summary in the terminal using CLI styling.
func PrintSummary(state ExecutionState, report SummaryReport) {
	fmt.Println()
	fmt.Println("\033[90m────────────────────────────────────────────────────────\033[0m")
	fmt.Println("\033[1;36m🤖 AI Summary\033[0m")
	fmt.Println("\033[90m────────────────────────────────────────────────────────\033[0m")
	fmt.Println()

	printSection("Execution Overview", report.ExecutionOverview)

	if state.Success {
		printSection("Infrastructure Created", report.Infrastructure)
		printSection("Validation Results", report.ValidationResults)
		printSection("Pipeline Stages", report.PipelineStages)
		printSection("Key Learnings", report.KeyLearnings)
		printSection("Recommendations", report.Recommendations)
	} else {
		printSection("Successful Stages", report.SuccessfulStages)
		printSection("Failed Stage", report.FailedStage)
		printSection("Skipped Stages", report.SkippedStages)
		printSection("Infrastructure State", report.InfrastructureState)
		printSection("Pipeline Stages", report.PipelineStages)
		printSection("Key Learnings", report.KeyLearnings)
		printSection("Recovery Steps", report.RecoverySteps)
		printSection("Recommendations", report.Recommendations)
		if report.OverallStatus != "" {
			printSection("Overall Pipeline Status", report.OverallStatus)
		}
	}

	fmt.Println("\033[90m────────────────────────────────────────────────────────\033[0m")
	fmt.Println()
}

func printSection(title, body string) {
	body = trimBody(body)
	if body == "" {
		return
	}
	fmt.Printf("\033[1m%s\033[0m\n", title)
	fmt.Printf("   %s\n\n", indentBody(body))
}

func trimBody(s string) string {
	if s == "" {
		return ""
	}
	return s
}

func indentBody(body string) string {
	lines := splitLines(body)
	if len(lines) <= 1 {
		return body
	}
	var out string
	for i, line := range lines {
		if i > 0 {
			out += "\n   "
		}
		out += line
	}
	return out
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
