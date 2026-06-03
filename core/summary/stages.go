package summary

import (
	"strings"
)

var stageDisplayNames = map[string]string{
	"Preflight":                           "Preflight",
	"Registry":                            "Registry",
	"Kind Cluster":                        "Kind Cluster",
	"Registry Bridge":                     "Registry Bridge",
	"Jenkins":                             "Jenkins",
	"Pipeline Job":                        "Pipeline Job",
	"Starting Registry":                   "Registry",
	"Creating empty Kind cluster":         "Kind Cluster",
	"Bridging Registry and Kind Networks": "Registry Bridge",
	"Booting Jenkins Container":           "Jenkins",
	"Framework Detection":                 "Framework Detection",
	"Scaffolding Generation":              "Scaffolding",
	"Port Forward":                        "Tunnel",
}

var plannedStageOrder = map[string][]string{
	"prep-ci": {"Preflight", "Registry", "Kind Cluster", "Registry Bridge", "Jenkins", "Pipeline Job"},
	"init":    {"Framework Detection", "Scaffolding Generation"},
}

func displayStageName(name string) string {
	if d, ok := stageDisplayNames[name]; ok {
		return d
	}
	return name
}

// formatStagesBrief shows high-level stages only (no install sub-step replay).
func formatStagesBrief(state ExecutionState) string {
	if planned, ok := plannedStageOrder[commandShort(state.Command)]; ok {
		return formatPlannedStages(state, planned)
	}
	return formatStagesFiltered(state)
}

func formatPlannedStages(state ExecutionState, planned []string) string {
	var b strings.Builder
	for _, plan := range planned {
		st, msg := resolvePlannedStage(state, plan)
		if st == "" {
			continue
		}
		icon := stageIcon(st)
		line := icon + " " + displayStageName(plan)
		if st == StageFailed && msg != "" {
			line += " — " + truncate(msg, 50)
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String())
}

func resolvePlannedStage(state ExecutionState, plan string) (StageStatus, string) {
	var match *StageRecord
	planLower := strings.ToLower(plan)
	for i := range state.Stages {
		s := &state.Stages[i]
		nameLower := strings.ToLower(s.Name)
		if s.Name == plan || nameLower == planLower ||
			strings.Contains(nameLower, planLower) ||
			strings.Contains(planLower, nameLower) {
			if match == nil || stagePriority(s.Status) > stagePriority(match.Status) {
				match = s
			}
		}
	}
	if match == nil {
		return "", ""
	}
	return match.Status, match.Message
}

func stagePriority(st StageStatus) int {
	switch st {
	case StageFailed:
		return 4
	case StageWarning:
		return 3
	case StageSuccess:
		return 2
	case StageSkipped:
		return 1
	default:
		return 0
	}
}

func formatStagesFiltered(state ExecutionState) string {
	var b strings.Builder
	n := 0
	for _, s := range state.Stages {
		if s.Status == StageSkipped && s.Message == "" {
			continue
		}
		if isGranularStep(s.Name) && s.Status == StageSuccess {
			continue
		}
		icon := stageIcon(s.Status)
		label := displayStageName(s.Name)
		line := icon + " " + label
		if s.Status == StageFailed && s.Message != "" {
			line += " — " + truncate(s.Message, 50)
		}
		b.WriteString(line + "\n")
		n++
		if n >= 6 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func isGranularStep(name string) bool {
	lower := strings.ToLower(name)
	keywords := []string{
		"installing", "injecting", "checking", "applying", "executing",
		"booting", "starting registry", "creating empty", "bridging",
	}
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
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
	seen := make(map[string]bool)
	var out []string
	for _, s := range state.Stages {
		if s.Status != StageSuccess {
			continue
		}
		label := displayStageName(s.Name)
		if seen[label] {
			continue
		}
		seen[label] = true
		out = append(out, label)
	}
	return out
}

func SkippedStageNames(state ExecutionState) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range state.Stages {
		if s.Status != StageSkipped {
			continue
		}
		label := displayStageName(s.Name)
		if seen[label] {
			continue
		}
		seen[label] = true
		out = append(out, label)
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
