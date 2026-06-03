package summary

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const markdownFile = "ai-summary.md"

// GenerateExecutionSummary builds, displays, and persists the summary (best-effort, never panics).
func GenerateExecutionSummary(state ExecutionState) {
	defer func() { recover() }()

	mu.Lock()
	once := &summaryOnce
	done := summaryDone
	mu.Unlock()

	once.Do(func() {
		if state.Command == "" {
			state.Command = lastCommandName
		}
		if state.Command == "" {
			return
		}

		fmt.Println()
		fmt.Println("\033[1;36m🤖 Generating AI Summary...\033[0m")

		report := generateReport(state)
		PrintSummary(state, report)

		md := report.RawMarkdown
		if md == "" {
			md = buildMarkdown(state, report)
		}
		path := writeMarkdownFile(md)

		mu.Lock()
		summaryDone = true
		mu.Unlock()

		if path != "" {
			fmt.Printf("\033[90m   Summary saved to %s\033[0m\n", path)
		}
		fmt.Println()
	})

	_ = done
}

func writeMarkdownFile(content string) string {
	if content == "" {
		content = "# AI Execution Summary\n\n**Generated:** " + time.Now().Format(time.RFC3339) + "\n\n_No summary content generated._\n"
	}
	cwd, err := os.Getwd()
	if err != nil {
		_ = os.WriteFile(markdownFile, []byte(content), 0644)
		return markdownFile
	}
	path := filepath.Join(cwd, markdownFile)
	_ = os.WriteFile(path, []byte(content), 0644)
	return path
}

// GenerateAndFinish completes the session and prints the summary.
func GenerateAndFinish(success bool) {
	state := Finish(success)
	GenerateExecutionSummary(state)
}

// FlushPending runs summary if a session is still open (cobra PostRun safety net).
func FlushPending() {
	mu.Lock()
	if summaryDone || session == nil {
		mu.Unlock()
		return
	}
	ok := runSuccess
	mu.Unlock()
	GenerateAndFinish(ok)
}

// GenerateAfterFailure runs summary after AI Analysis.
func GenerateAfterFailure() {
	MarkFailed()
	MarkRemainingSkipped()
	state := Finish(false)
	GenerateExecutionSummary(state)
}
