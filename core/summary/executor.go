package summary

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const markdownFile = "ai-summary.md"

// GenerateExecutionSummary builds, displays, and persists the summary (best-effort).
func GenerateExecutionSummary(state ExecutionState) {
	defer func() { recover() }()

	mu.Lock()
	once := &summaryOnce
	mu.Unlock()

	once.Do(func() {
		if state.Command == "" {
			state.Command = lastCommandName
		}
		if state.Command == "" {
			writeMarkdownFile(minimalMarkdown("unknown"))
			return
		}

		fmt.Println("\n\033[1;36m🤖 AI Summary\033[0m")

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
			fmt.Printf("\033[90m  → %s\033[0m\n", path)
		}
	})
}

func minimalMarkdown(cmd string) string {
	return "# AI Execution Summary\n\n**Command:** " + cmd + "\n\n**When:** " +
		time.Now().Format("2006-01-02 15:04:05") + "\n\n_No summary content._\n"
}

func writeMarkdownFile(content string) string {
	if content == "" {
		content = minimalMarkdown(lastCommandName)
	}
	cwd, err := os.Getwd()
	path := markdownFile
	if err == nil {
		path = filepath.Join(cwd, markdownFile)
	}
	_ = os.WriteFile(path, []byte(content), 0644)
	return path
}

// GenerateAndFinish completes the session and prints the summary.
func GenerateAndFinish(success bool) {
	state := Finish(success)
	GenerateExecutionSummary(state)
}

// FlushPending runs summary if a session is still open.
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
