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

// PrintSummary renders the executive terminal summary using command-specific templates.
func PrintSummary(state ExecutionState, report SummaryReport) {
	layout := layoutFor(state)
	ids := terminalSections(layout, state.Success)

	fmt.Println()
	fmt.Println("\033[90m────────────────────────────────────────────────────────\033[0m")
	fmt.Println("\033[1;36m🤖 AI Summary\033[0m")
	fmt.Println("\033[90m────────────────────────────────────────────────────────\033[0m")
	fmt.Println()

	for _, id := range ids {
		body := sectionBody(id, report)
		if id == secResult {
			body = report.ExecutionResult
		}
		printSection(sectionTitle(id), body)
	}

	fmt.Println("\033[90m────────────────────────────────────────────────────────\033[0m")
	fmt.Println()
}

func printSection(title, body string) {
	body = stringsTrim(body)
	if body == "" {
		return
	}
	fmt.Printf("\033[1m%s\033[0m\n", title)
	fmt.Printf("   %s\n\n", indentBody(body))
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
