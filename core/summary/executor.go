package summary

import (
	"os"
	"path/filepath"
)

const markdownFile = "ai-summary.md"

// GenerateExecutionSummary builds, displays, and persists the execution summary.
// Best-effort: never panics and never returns an error to callers.
func GenerateExecutionSummary(state ExecutionState) {
	defer func() { recover() }()

	if state.Command == "" {
		return
	}

	report := generateReport(state)
	PrintSummary(state, report)
	writeMarkdownFile(report.RawMarkdown)
}

func writeMarkdownFile(content string) {
	if content == "" {
		content = "# AI Execution Summary\n\n_Summary generation produced no content._\n"
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	path := filepath.Join(cwd, markdownFile)
	_ = os.WriteFile(path, []byte(content), 0644)
}

// GenerateAndFinish is a convenience for successful command completion.
func GenerateAndFinish(success bool) {
	state := Finish(success)
	GenerateExecutionSummary(state)
}

// GenerateAfterFailure runs summary after AI Analysis on failure paths.
func GenerateAfterFailure() {
	MarkRemainingSkipped()
	state := Finish(false)
	GenerateExecutionSummary(state)
}
