package ai

import "testing"

func TestMatchKnownFailure_DockerDaemon(t *testing.T) {
	ctx := FailureContext{
		Command:    "pipeline prep-ci",
		Stage:      "Docker Build",
		Error:      "Cannot connect to the Docker daemon",
		RecentLogs: []string{"error during connect: open //./pipe/docker_engine: The system cannot find the file specified."},
	}

	result, ok := matchKnownFailure(ctx)
	if !ok {
		t.Fatal("expected local match for docker daemon error")
	}
	if result.Source != "local" {
		t.Fatalf("expected local source, got %q", result.Source)
	}
	if result.Issue == "" || result.RootCause == "" {
		t.Fatal("expected populated diagnosis")
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
