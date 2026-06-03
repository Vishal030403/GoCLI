package summary

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateFallback_SuccessPrepCI(t *testing.T) {
	state := ExecutionState{
		Command:   "pipeline prep-ci",
		Success:   true,
		StartTime: time.Now().Add(-2 * time.Minute),
		EndTime:   time.Now(),
		Duration:  2 * time.Minute,
		Stages: []StageRecord{
			{Name: "Preflight", Status: StageSuccess},
			{Name: "Registry", Status: StageSuccess},
			{Name: "Jenkins", Status: StageSuccess},
		},
		Infrastructure: []InfrastructureItem{
			{Name: "Registry", Detail: "127.0.0.1:5001"},
			{Name: "Kind Cluster", Detail: "ephemeral-test"},
		},
	}
	report := generateFallback(state)
	if !strings.Contains(report.ExecutionOverview, "prep-ci") {
		t.Fatalf("expected prep-ci in overview: %q", report.ExecutionOverview)
	}
	if report.KeyLearnings == "" {
		t.Fatal("expected learnings")
	}
}

func TestGenerateExecutionSummary_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked: %v", r)
		}
	}()
	GenerateExecutionSummary(ExecutionState{Command: "pipeline init", Success: true})
}
