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
		ids = append(terminalSections(layout, true), layout.markdownExtra...)
	} else {
		ids = append(terminalSections(layout, false), layout.markdownExtra...)
	}

	var b strings.Builder
	b.WriteString("# AI Execution Summary\n\n")
	b.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))
	b.WriteString("**Command:** " + state.Command + "\n\n")
	b.WriteString("**Duration:** " + formatDuration(state) + "\n\n")

	seen := map[sectionID]bool{}
	all := append([]sectionID{secResult}, ids...)
	for _, id := range all {
		if seen[id] {
			continue
		}
		seen[id] = true
		body := sectionBody(id, r)
		if id == secResult {
			body = r.ExecutionResult
		}
		writeMDSection(&b, sectionTitle(id), body)
	}

	if warningStageCount(state) > 0 {
		var w strings.Builder
		for _, msg := range state.Warnings {
			w.WriteString("- " + msg + "\n")
		}
		for _, s := range state.Stages {
			if s.Status == StageWarning {
				fmt.Fprintf(&w, "- %s\n", displayStageName(s.Name))
			}
		}
		writeMDSection(&b, "Warnings", strings.TrimSpace(w.String()))
	}

	if len(state.Errors) > 0 {
		var e strings.Builder
		for _, msg := range state.Errors {
			e.WriteString("- " + msg + "\n")
		}
		writeMDSection(&b, "Failures", strings.TrimSpace(e.String()))
	}

	if r.DeveloperNotes != "" {
		writeMDSection(&b, "Developer Notes", r.DeveloperNotes)
	}

	return b.String()
}

func writeMDSection(b *strings.Builder, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	b.WriteString("## " + title + "\n\n")
	b.WriteString(body + "\n\n")
}
