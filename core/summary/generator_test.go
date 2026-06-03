package summary

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateFallback_NoFalseWarnings(t *testing.T) {
	state := ExecutionState{
		Command: "pipeline prep-ci",
		Success: true,
		Stages: []StageRecord{
			{Name: "Starting Registry", Status: StageSuccess},
			{Name: "Bridging Registry and Kind Networks", Status: StageSuccess},
			{Name: "Booting Jenkins Container", Status: StageSuccess},
		},
	}
	report := sanitizeReport(state, generateFallback(state))
	combined := report.ExecutionOverview + report.PipelineStages + report.ValidationResults
	if strings.Contains(strings.ToLower(combined), "warning") {
		t.Fatalf("expected no warning language when none occurred: %q", combined)
	}
}

func TestSanitizeReport_StripsWarningsWhenZero(t *testing.T) {
	state := ExecutionState{Command: "pipeline prep-ci", Success: true}
	r := SummaryReport{ExecutionOverview: "Registry completed with warnings"}
	out := sanitizeReport(state, r)
	if strings.Contains(strings.ToLower(out.ExecutionOverview), "warning") {
		t.Fatal("sanitize should strip invented warnings")
	}
}

func TestFormatStagesBrief(t *testing.T) {
	state := ExecutionState{
		Command: "pipeline prep-ci",
		Stages: []StageRecord{
			{Name: "Preflight", Status: StageSuccess},
			{Name: "Registry", Status: StageSuccess},
		},
	}
	s := formatStagesBrief(state)
	if !strings.Contains(s, "✓") {
		t.Fatalf("expected checkmarks: %q", s)
	}
}

func TestFormatStagesBrief_PrepCI_HidesGranular(t *testing.T) {
	state := ExecutionState{
		Command: "pipeline prep-ci",
		Stages: []StageRecord{
			{Name: "Preflight", Status: StageSuccess},
			{Name: "Registry", Status: StageSuccess},
			{Name: "Installing Docker CLI inside Jenkins", Status: StageSuccess},
			{Name: "Jenkins", Status: StageSuccess},
		},
	}
	s := formatStagesBrief(state)
	if strings.Contains(s, "Installing Docker") {
		t.Fatalf("granular step should be hidden: %q", s)
	}
	if !strings.Contains(s, "Jenkins") {
		t.Fatalf("expected Jenkins in brief stages: %q", s)
	}
}

func TestGenerateExecutionSummary_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked: %v", r)
		}
	}()
	ResetSummaryGuard()
	GenerateExecutionSummary(ExecutionState{
		Command:   "pipeline init",
		Success:   true,
		StartTime: time.Now(),
		EndTime:   time.Now(),
	})
}
