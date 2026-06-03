package summary

import (
	"fmt"
	"strings"
)

func terminalSections(layout templateLayout, success bool) []sectionID {
	if success {
		return layout.terminalSuccess
	}
	return layout.terminalFailure
}

// PrintSummary renders a compact terminal summary.
func PrintSummary(state ExecutionState, report SummaryReport) {
	report = compactForTerminal(report)
	layout := layoutFor(state)
	ids := terminalSections(layout, state.Success)

	fmt.Println()
	fmt.Println("\033[90m────────────────────────────\033[0m")
	fmt.Println("\033[1;36m🤖 AI Summary\033[0m")
	fmt.Println("\033[90m────────────────────────────\033[0m")

	for _, id := range ids {
		body := sectionBody(id, report)
		if id == secResult {
			body = report.ExecutionResult
		}
		printSection(sectionTitle(id), body)
	}

	fmt.Println("\033[90m────────────────────────────\033[0m")
}

func printSection(title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	if !strings.Contains(body, "\n") {
		fmt.Printf("  \033[1m%s:\033[0m %s\n", title, body)
		return
	}
	fmt.Printf("  \033[1m%s:\033[0m\n", title)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			fmt.Printf("     %s\n", line)
		}
	}
}
