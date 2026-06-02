package policy

import (
	"fmt"
	"strings"

	"pipeline-cli/core"
	"pipeline-cli/core/ai"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[1;31m"
	colorGreen  = "\033[1;32m"
	colorYellow = "\033[1;33m"
	colorCyan   = "\033[1;36m"
	colorBold   = "\033[1m"
)

// PrintReport prints a formatted policy check report to stdout.
// Emits AI recommendations for warnings and analysis for error-severity failures.
// Returns true if any error-severity policy failed (so the caller can exit 1).
func PrintReport(results []PolicyResult) bool {
	separator := strings.Repeat("─", 52)

	fmt.Printf("\n%s\n", separator)
	fmt.Printf("  %sPolicy Check Results%s\n", colorBold, colorReset)
	fmt.Printf("%s\n", separator)

	passed := 0
	warnings := 0
	failed := 0
	anyErrorFailed := false
	var warningItems []ai.WarningItem

	for _, r := range results {
		if r.Passed {
			passed++
			fmt.Printf("%s✅ PASS%s  [%-12s] %s\n", colorGreen, colorReset, r.Category, r.PolicyName)
		} else if r.Severity == "warning" {
			warnings++
			fmt.Printf("%s⚠️  WARN%s  [%-12s] %s\n", colorYellow, colorReset, r.Category, r.PolicyName)
			printFindings(r)
			warningItems = append(warningItems, toWarningItem(r))
		} else {
			failed++
			anyErrorFailed = true
			fmt.Printf("%s❌ FAIL%s  [%-12s] %s\n", colorRed, colorReset, r.Category, r.PolicyName)
			printFindings(r)
		}
	}

	fmt.Printf("%s\n", separator)
	fmt.Printf("  %s%d passed%s   %s%d warnings%s   %s%d failed%s\n",
		colorGreen, passed, colorReset,
		colorYellow, warnings, colorReset,
		colorRed, failed, colorReset,
	)
	fmt.Printf("%s\n\n", separator)

	if len(warningItems) > 0 {
		ai.PrintRecommendations(ai.RecommendForWarnings(warningItems))
	}

	if anyErrorFailed {
		for _, r := range results {
			if !r.Passed && r.Severity != "warning" {
				ai.HandleFailure(buildPolicyFailureContext(r))
				break
			}
		}
	}

	return anyErrorFailed
}

func toWarningItem(r PolicyResult) ai.WarningItem {
	item := ai.WarningItem{
		PolicyName: r.PolicyName,
		Category:   r.Category,
		Message:    r.Message,
	}
	for _, f := range r.Findings {
		if f.File != "" && f.Line > 0 {
			item.Findings = append(item.Findings, fmt.Sprintf("%s:%d — %s", f.File, f.Line, f.Detail))
		} else if f.File != "" {
			item.Findings = append(item.Findings, fmt.Sprintf("%s — %s", f.File, f.Detail))
		} else {
			item.Findings = append(item.Findings, f.Detail)
		}
	}
	return item
}

func buildPolicyFailureContext(r PolicyResult) ai.FailureContext {
	cmd := core.CommandName
	if cmd == "" {
		cmd = "pipeline policy check"
	}

	errMsg := r.Message
	if errMsg == "" && len(r.Findings) > 0 {
		errMsg = r.Findings[0].Detail
	}
	if errMsg == "" {
		errMsg = "policy check failed: " + r.PolicyName
	}

	var logs []string
	for _, f := range r.Findings {
		if f.File != "" {
			logs = append(logs, fmt.Sprintf("%s — %s", f.File, f.Detail))
		} else {
			logs = append(logs, f.Detail)
		}
	}
	if r.Message != "" {
		logs = append(logs, r.Message)
	}

	return ai.FailureContext{
		Command:     cmd,
		Stage:       "Policy Check",
		FailedStage: r.PolicyName,
		Error:       errMsg,
		ExitCode:    1,
		RecentLogs:  logs,
	}
}

func printFindings(r PolicyResult) {
	if len(r.Findings) == 0 {
		if r.Message != "" {
			fmt.Printf("         → %s\n", r.Message)
		}
		return
	}
	for _, f := range r.Findings {
		if f.File != "" && f.Line > 0 {
			fmt.Printf("         → %s:%d — %s\n", f.File, f.Line, f.Detail)
		} else if f.File != "" {
			fmt.Printf("         → %s — %s\n", f.File, f.Detail)
		} else {
			fmt.Printf("         → %s\n", f.Detail)
		}
	}
}

// CategoryLabel wraps a category string in cyan for the list table.
func CategoryLabel(cat string) string {
	return fmt.Sprintf("%s[%s]%s", colorCyan, cat, colorReset)
}
