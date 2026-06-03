package summary

import (
	"strings"
)

// stageDisplayNames maps internal step names to executive-friendly labels.
var stageDisplayNames = map[string]string{
	"Preflight":                              "Preflight Validation",
	"Registry":                               "Local Registry Setup",
	"Kind Cluster":                           "Kubernetes Cluster Creation",
	"Registry Bridge":                        "Registry Network Bridge",
	"Jenkins":                                "Jenkins Deployment",
	"Pipeline Job":                           "Pipeline Bootstrap",
	"Starting Registry":                      "Local Registry Setup",
	"Creating empty Kind cluster":            "Kubernetes Cluster Creation",
	"Bridging Registry and Kind Networks":    "Registry Network Bridge",
	"Booting Jenkins Container":              "Jenkins Deployment",
	"Framework Detection":                    "Project Detection",
	"Scaffolding Generation":                 "Scaffolding Generation",
	"Port Forward":                           "Application Tunnel",
}

func displayStageName(name string) string {
	if d, ok := stageDisplayNames[name]; ok {
		return d
	}
	return name
}

// formatStagesConcise builds checkmark lines without replaying raw logs.
func formatStagesConcise(state ExecutionState) string {
	var b strings.Builder
	for _, s := range state.Stages {
		if s.Status == StageSkipped && s.Message == "" {
			continue
		}
		icon := stageIcon(s.Status)
		label := displayStageName(s.Name)
		switch s.Status {
		case StageFailed:
			b.WriteString(icon + " " + label)
			if s.Message != "" {
				b.WriteString(" — " + truncate(s.Message, 60))
			}
		case StageWarning:
			b.WriteString(icon + " " + label + " (review)")
		default:
			b.WriteString(icon + " " + label)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func stageIcon(status StageStatus) string {
	switch status {
	case StageSuccess:
		return "✓"
	case StageFailed:
		return "✗"
	case StageSkipped:
		return "○"
	case StageWarning:
		return "!"
	default:
		return "•"
	}
}

func SuccessfulStageNames(state ExecutionState) []string {
	var out []string
	for _, s := range state.Stages {
		if s.Status == StageSuccess {
			out = append(out, displayStageName(s.Name))
		}
	}
	return out
}

func SkippedStageNames(state ExecutionState) []string {
	var out []string
	for _, s := range state.Stages {
		if s.Status == StageSkipped {
			out = append(out, displayStageName(s.Name))
		}
	}
	return out
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func warningStageCount(state ExecutionState) int {
	n := len(state.Warnings)
	for _, s := range state.Stages {
		if s.Status == StageWarning {
			n++
		}
	}
	return n
}
