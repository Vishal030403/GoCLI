package ai

import (
	"regexp"
	"strings"
)

const maxRecentLogLines = 30

var (
	jenkinsStageRe   = regexp.MustCompile(`(?i)(?:Stage|stage)\s+['"]?([^'"\n]+)['"]?\s+(?:failed|Skipped due to earlier failure)`)
	jenkinsSkippedRe = regexp.MustCompile(`(?i)Stage\s+['"]?([^'"\n]+)['"]?\s+Skipped due to earlier failure`)
	jenkinsFailedRe  = regexp.MustCompile(`(?i)(?:ERROR:|FAILED|Failure|Build step '[^']+' marked build as failure)`)
	k8sPodErrorRe    = regexp.MustCompile(`(?i)(ImagePullBackOff|CrashLoopBackOff|ErrImagePull|CreateContainerConfigError|Back-off restarting)`)
)

// ExtractRecentLogs returns the last N relevant log lines, prioritising error lines.
func ExtractRecentLogs(raw string, limit int) []string {
	if limit <= 0 {
		limit = maxRecentLogLines
	}
	if raw == "" {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) <= limit {
		return filterNoise(lines)
	}

	// Collect error-ish lines first, then fill with trailing context.
	var errorLines, tail []string
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isRelevantErrorLine(trimmed) {
			errorLines = append(errorLines, trimmed)
		}
		if i >= len(lines)-limit/2 {
			tail = append(tail, trimmed)
		}
	}

	merged := dedupeLines(append(errorLines, tail...))
	if len(merged) > limit {
		merged = merged[len(merged)-limit:]
	}
	return merged
}

// EnrichFromJenkinsLog extracts Jenkins-specific fields from log content.
func EnrichFromJenkinsLog(ctx *FailureContext, raw string) {
	if ctx == nil || raw == "" {
		return
	}

	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "failure"):
		ctx.FinalStatus = "FAILURE"
	case strings.Contains(lower, "aborted"):
		ctx.FinalStatus = "ABORTED"
	case strings.Contains(lower, "unstable"):
		ctx.FinalStatus = "UNSTABLE"
	}

	for _, m := range jenkinsSkippedRe.FindAllStringSubmatch(raw, -1) {
		if len(m) > 1 {
			ctx.SkippedStages = appendUnique(ctx.SkippedStages, strings.TrimSpace(m[1]))
		}
	}

	// Failed stage: last explicit stage failure wins.
	for _, m := range jenkinsStageRe.FindAllStringSubmatch(raw, -1) {
		if len(m) > 1 && strings.Contains(strings.ToLower(m[0]), "failed") {
			ctx.FailedStage = strings.TrimSpace(m[1])
		}
	}
	if ctx.FailedStage == "" {
		ctx.FailedStage = detectJenkinsStageFromError(raw)
	}
}

// BuildFailureContext assembles a FailureContext from execution metadata and raw output.
func BuildFailureContext(command, stage, errMsg string, exitCode int, rawOutput string) FailureContext {
	logs := ExtractRecentLogs(rawOutput, maxRecentLogLines)
	ctx := FailureContext{
		Command:    command,
		Stage:      stage,
		Error:      errMsg,
		ExitCode:   exitCode,
		RecentLogs: logs,
	}
	EnrichFromJenkinsLog(&ctx, rawOutput)
	if ctx.FailedStage == "" {
		ctx.FailedStage = stage
	}
	return ctx
}

func detectJenkinsStageFromError(raw string) string {
	lower := strings.ToLower(raw)
	stages := []struct {
		keyword string
		name    string
	}{
		{"docker build", "Docker Build"},
		{"npm test", "Unit Tests"},
		{"pytest", "Unit Tests"},
		{"checkov", "Security Scan"},
		{"trivy", "Security Scan"},
		{"docker push", "Registry Push"},
		{"kubectl apply", "Deployment"},
		{"helm upgrade", "Deployment"},
		{"kustomize", "Deployment"},
	}
	for _, s := range stages {
		if strings.Contains(lower, s.keyword) {
			return s.name
		}
	}
	return ""
}

func isRelevantErrorLine(line string) bool {
	lower := strings.ToLower(line)
	keywords := []string{
		"error", "failed", "failure", "fatal", "exception",
		"denied", "not found", "cannot", "unable", "backoff",
		"exit code", "no such", "permission", "timeout",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return jenkinsFailedRe.MatchString(line) || k8sPodErrorRe.MatchString(line)
}

func filterNoise(lines []string) []string {
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func dedupeLines(lines []string) []string {
	seen := make(map[string]bool, len(lines))
	var out []string
	for _, l := range lines {
		if !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out
}

func appendUnique(list []string, item string) []string {
	for _, existing := range list {
		if existing == item {
			return list
		}
	}
	return append(list, item)
}

func corpus(ctx FailureContext) string {
	var sb strings.Builder
	sb.WriteString(ctx.Error)
	sb.WriteByte('\n')
	for _, l := range ctx.RecentLogs {
		sb.WriteString(l)
		sb.WriteByte('\n')
	}
	return sb.String()
}
