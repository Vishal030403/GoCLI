package ai

import "testing"

func TestEnrichFromJenkinsLog_SkippedStages(t *testing.T) {
	ctx := FailureContext{}
	raw := `Stage 'Docker Build' failed
ERROR: package.json not found
Stage 'Unit Tests' Skipped due to earlier failure
Finished: FAILURE`
	EnrichFromJenkinsLog(&ctx, raw)
	if ctx.FailedStage == "" {
		t.Fatal("expected failed stage")
	}
	if len(ctx.SkippedStages) == 0 {
		t.Fatal("expected skipped stages")
	}
	if ctx.FinalStatus != "FAILURE" {
		t.Fatalf("expected FAILURE, got %q", ctx.FinalStatus)
	}
}

func TestExtractRecentLogs_PrioritizesErrors(t *testing.T) {
	raw := "info line\nanother info\nERROR: package.json not found\nfinal line"
	lines := ExtractRecentLogs(raw, 10)
	found := false
	for _, l := range lines {
		if l == "ERROR: package.json not found" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected error line in recent logs, got %v", lines)
	}
}

func TestRecommendForWarnings_HealthEndpoint(t *testing.T) {
	recs := RecommendForWarnings([]WarningItem{{PolicyName: "health-endpoint", Message: "No health endpoint found"}})
	if len(recs) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recs))
	}
	if recs[0].SuggestedFix == "" {
		t.Fatal("expected suggested fix")
	}
}
