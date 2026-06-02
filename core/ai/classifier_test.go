package ai

import (
	"strings"
	"testing"
)

func TestClassify_ConnectionRefused_CrashLoop_NotRegistry(t *testing.T) {
	ctx := DiagnosticContext{
		Command:       "pipeline tunnel",
		Stage:         "Port Forward",
		Error:         "failed to connect to localhost:8080: connection refused",
		PodStatus:     "CrashLoopBackOff",
		PodNamespace:  "myapp-ns",
		PodName:       "myapp-abc123",
		ServiceExists: true,
		Evidence: []string{
			"Pod myapp-ns/myapp-abc123: status=CrashLoopBackOff, restarts=5",
			"Service myapp-ns/myapp: exists",
		},
	}

	result, conf, ok := Classify(ctx)
	if !ok {
		t.Fatal("expected classification")
	}
	if conf != ConfidenceHigh {
		t.Fatalf("expected high confidence, got %s", conf)
	}
	if strings.Contains(strings.ToLower(result.Issue), "registry") {
		t.Fatalf("incorrect registry diagnosis: %q", result.Issue)
	}
	issueLower := strings.ToLower(result.Issue)
	if !strings.Contains(issueLower, "crash") && !strings.Contains(issueLower, "application") {
		t.Fatalf("expected application/crash diagnosis, got %q", result.Issue)
	}
}

func TestClassify_RegistryUnreachable(t *testing.T) {
	ctx := DiagnosticContext{
		Command:           "pipeline prep-ci",
		Stage:             "Registry Push",
		Error:             "dial tcp 127.0.0.1:5001: connect: connection refused",
		RegistryReachable: false,
		Evidence:          []string{"Registry HTTP check (127.0.0.1:5001): unreachable"},
	}

	result, conf, ok := Classify(ctx)
	if !ok {
		t.Fatal("expected classification")
	}
	if conf != ConfidenceHigh {
		t.Fatalf("expected high confidence, got %s", conf)
	}
	if !strings.Contains(strings.ToLower(result.Issue), "registry") {
		t.Fatalf("expected registry issue, got %q", result.Issue)
	}
}

func TestClassify_Tunnel_ServiceExists_CrashLoop(t *testing.T) {
	ctx := DiagnosticContext{
		Command:           "pipeline tunnel",
		Stage:             "Port Forward",
		Error:             "connection refused",
		PodStatus:         "CrashLoopBackOff",
		ServiceExists:     true,
		PortForwardFailed: true,
	}

	result, _, ok := Classify(ctx)
	if !ok {
		t.Fatal("expected classification")
	}
	if strings.Contains(strings.ToLower(result.RootCause), "registry") {
		t.Fatalf("should not blame registry: %q", result.RootCause)
	}
}

func TestMatchKnownFailure_DockerDaemon(t *testing.T) {
	ctx := DiagnosticContext{
		Command:        "pipeline prep-ci",
		Stage:          "Docker Build",
		Error:          "Cannot connect to the Docker daemon",
		DockerDaemonOK: false,
		RecentLogs:     []string{"error during connect: open //./pipe/docker_engine"},
		Evidence:       []string{"Docker daemon: unreachable"},
	}

	result, ok := matchKnownFailure(ctx)
	if !ok {
		t.Fatal("expected local match for docker daemon error")
	}
	if result.Source != "local" {
		t.Fatalf("expected local source, got %q", result.Source)
	}
}
