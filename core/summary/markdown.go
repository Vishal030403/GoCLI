package summary

import (
	"fmt"
	"strings"
	"time"
)

func buildMarkdown(state ExecutionState, r SummaryReport) string {
	layout := layoutFor(state)
	ids := layout.markdownExtra
	if state.Success {
		ids = append(terminalSections(layout, true), ids...)
	} else {
		ids = append(terminalSections(layout, false), ids...)
	}

	var b strings.Builder
	b.WriteString("# AI Execution Summary\n\n")
	b.WriteString(fmt.Sprintf("**When:** %s  \n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("**Command:** %s  \n", state.Command))
	b.WriteString(fmt.Sprintf("**Result:** %s  \n\n", buildExecutionResult(state)))

	seen := map[sectionID]bool{}
	for _, id := range append([]sectionID{secStages}, ids...) {
		if seen[id] || id == secResult {
			continue
		}
		seen[id] = true
		body := sectionBody(id, r)
		if id == secStages && body == "" {
			body = formatStagesBrief(state)
		}
		writeMDSection(&b, sectionTitle(id), body)
	}

	if warningStageCount(state) > 0 && len(state.Warnings) > 0 {
		writeMDSection(&b, "Warnings", strings.Join(state.Warnings, "\n"))
	}

	return strings.TrimSpace(b.String()) + "\n"
}

func writeMDSection(b *strings.Builder, title, body string) {
	body = strings.TrimSpace(clampLines(body, 12))
	if body == "" {
		return
	}
	b.WriteString("## " + title + "\n\n")
	b.WriteString(body + "\n\n")
}
