package ai

import (
	"strings"
)

// Classify produces a diagnosis from verified evidence. The second return is whether
// the result is authoritative enough to skip Gemini.
func Classify(ctx DiagnosticContext) (AnalysisResult, Confidence, bool) {
	text := strings.ToLower(corpus(ctx))

	// --- High confidence: verified Kubernetes pod state ---
	if podCrash := strings.Contains(strings.ToLower(ctx.PodStatus), "crashloopbackoff"); podCrash {
		return crashLoopResult(ctx), ConfidenceHigh, true
	}
	if strings.Contains(text, "crashloopbackoff") && ctx.PodStatus == "" {
		// Log mentions crash loop but kubectl did not confirm — medium until verified
		if hasK8sEvidence(ctx) {
			return crashLoopResult(ctx), ConfidenceMedium, true
		}
	}

	if strings.Contains(text, "imagepullbackoff") || strings.Contains(text, "errimagepull") {
		if ctx.RegistryReachable {
			return imagePullWithRegistryUpResult(), ConfidenceHigh, true
		}
		if !ctx.RegistryReachable && hasRegistryEvidence(ctx) {
			return registryUnavailableResult(), ConfidenceHigh, true
		}
	}

	// --- Connection refused: context-aware (never assume registry alone) ---
	if strings.Contains(text, "connection refused") || strings.Contains(text, "connect: connection refused") {
		podCrash := strings.Contains(strings.ToLower(ctx.PodStatus), "crashloopbackoff")
		if podCrash || strings.Contains(strings.ToLower(ctx.PodStatus), "error") {
			return applicationUnreachableResult(ctx), ConfidenceHigh, true
		}
		if strings.Contains(strings.ToLower(ctx.Command), "tunnel") || ctx.PortForwardFailed {
			if ctx.ServiceExists {
				return tunnelBackendFailureResult(ctx), ConfidenceHigh, true
			}
			if ctx.ServiceExists == false && hasK8sEvidence(ctx) {
				return serviceMisconfigurationResult(ctx), ConfidenceMedium, true
			}
		}
		if isRegistryPort(text) && !ctx.RegistryReachable && hasRegistryEvidence(ctx) {
			return registryUnavailableResult(), ConfidenceHigh, true
		}
		if ctx.RegistryReachable && (strings.Contains(text, "localhost:808") || strings.Contains(text, "127.0.0.1:808")) {
			return applicationUnreachableResult(ctx), ConfidenceHigh, true
		}
	}

	// --- Docker daemon ---
	if !ctx.DockerDaemonOK && hasDockerEvidence(ctx) {
		if strings.Contains(text, "cannot connect to the docker daemon") ||
			strings.Contains(text, "error during connect") ||
			strings.Contains(text, "docker daemon") {
			if result, ok := matchPatternRule(ctx, "docker daemon"); ok {
				return result, ConfidenceHigh, true
			}
		}
	}

	// --- Jenkins / Docker build (package.json etc.) ---
	if strings.Contains(text, "package.json not found") {
		if result, ok := matchPatternRule(ctx, "package.json"); ok {
			result.Explanation = jenkinsDockerBuildExplanation(ctx)
			return result, ConfidenceHigh, true
		}
	}

	// --- Pattern rules only when evidence does not contradict ---
	if result, ok := matchEvidenceAwareRules(ctx); ok {
		conf := ConfidenceHigh
		if !hasStrongEvidence(ctx) {
			conf = ConfidenceMedium
		}
		return result, conf, true
	}

	return AnalysisResult{}, ConfidenceLow, false
}

func matchEvidenceAwareRules(ctx DiagnosticContext) (AnalysisResult, bool) {
	text := strings.ToLower(corpus(ctx))

	// Skip registry rule when connection refused targets app port, not registry
	skipRegistry := strings.Contains(text, "connection refused") &&
		(ctx.RegistryReachable || strings.Contains(text, "localhost:808") || strings.Contains(text, ":8080"))

	for _, rule := range knownFailureRules {
		if skipRegistry && rule.issue == "Local container registry is unavailable" {
			continue
		}
		for _, pattern := range rule.patterns {
			if !strings.Contains(text, strings.ToLower(pattern)) {
				continue
			}
			// Extra guards for ambiguous patterns
			if rule.issue == "Local container registry is unavailable" {
				if ctx.RegistryReachable {
					continue
				}
				if !isRegistryPort(text) && !strings.Contains(text, "registry") && !strings.Contains(text, "5001") {
					continue
				}
			}
			return AnalysisResult{
				Issue:       rule.issue,
				RootCause:   rule.root,
				Explanation: rule.explain,
				Impact:      rule.impact,
				Resolution:  rule.fix,
				Prevention:  rule.prevent,
				FixCommands: rule.commands,
				Source:      "local",
			}, true
		}
	}
	return AnalysisResult{}, false
}

func matchPatternRule(ctx DiagnosticContext, keyword string) (AnalysisResult, bool) {
	text := strings.ToLower(corpus(ctx))
	for _, rule := range knownFailureRules {
		if !strings.Contains(strings.ToLower(rule.issue), keyword) {
			continue
		}
		for _, pattern := range rule.patterns {
			if strings.Contains(text, strings.ToLower(pattern)) {
				return AnalysisResult{
					Issue:       rule.issue,
					RootCause:   rule.root,
					Explanation: rule.explain,
					Impact:      rule.impact,
					Resolution:  rule.fix,
					Prevention:  rule.prevent,
					FixCommands: rule.commands,
					Source:      "local",
				}, true
			}
		}
	}
	return AnalysisResult{}, false
}

func hasK8sEvidence(ctx DiagnosticContext) bool {
	for _, e := range ctx.Evidence {
		if strings.Contains(e, "kubectl get pods") || strings.Contains(e, "Pod ") {
			return true
		}
	}
	return false
}

func hasRegistryEvidence(ctx DiagnosticContext) bool {
	for _, e := range ctx.Evidence {
		if strings.Contains(e, "Registry") {
			return true
		}
	}
	return ctx.RegistryStatus != "" || ctx.RegistryReachable
}

func hasDockerEvidence(ctx DiagnosticContext) bool {
	for _, e := range ctx.Evidence {
		if strings.Contains(e, "Docker") {
			return true
		}
	}
	return true
}

func hasStrongEvidence(ctx DiagnosticContext) bool {
	return ctx.PodStatus != "" || ctx.RegistryReachable || ctx.RegistryStatus != "" || ctx.DockerDaemonOK
}

func isRegistryPort(text string) bool {
	return strings.Contains(text, "5001") ||
		strings.Contains(text, "127.0.0.1:500") ||
		strings.Contains(text, "local-registry") ||
		strings.Contains(text, "registry unavailable")
}

func crashLoopResult(ctx DiagnosticContext) AnalysisResult {
	explain := "The container starts but exits immediately. Kubernetes keeps restarting it (CrashLoopBackOff)."
	if ctx.PodName != "" {
		explain += " Verified pod: " + ctx.PodNamespace + "/" + ctx.PodName + "."
	}
	return AnalysisResult{
		Issue:       "Application crash inside the pod (CrashLoopBackOff)",
		RootCause:   "The application process inside the container is failing on startup — not the registry or tunnel itself.",
		Explanation: explain,
		Impact:      "The pod never becomes Ready. Port-forwards and health checks fail with connection refused.",
		Resolution: []string{
			"Inspect application logs: kubectl logs -n " + nsOrPlaceholder(ctx) + " " + podOrPlaceholder(ctx),
			"Check the previous crash: kubectl logs -n " + nsOrPlaceholder(ctx) + " " + podOrPlaceholder(ctx) + " --previous",
			"Verify CMD/ENTRYPOINT, environment variables, and listening port in the container.",
			"Run the image locally: docker run --rm -it <image> to reproduce the crash.",
		},
		Prevention:  "Add liveness/readiness probes and test the container image locally before deploy.",
		FixCommands: []string{"kubectl logs -n " + nsOrPlaceholder(ctx) + " " + podOrPlaceholder(ctx) + " --previous"},
		Source:      "local",
	}
}

func applicationUnreachableResult(ctx DiagnosticContext) AnalysisResult {
	return AnalysisResult{
		Issue:       "Application not accepting connections",
		RootCause:   "Something accepted the network path (service/tunnel) but nothing is listening on the target port inside the workload — typically because the app crashed or never bound to the port.",
		Explanation: "Connection refused on localhost:8080 (or similar) after port-forward usually means the pod is running the forward but the process inside is down or listening on a different port.",
		Impact:      "The tunnel or probe appears up while the application is unreachable.",
		Resolution: []string{
			"Confirm pod status: kubectl get pods -n " + nsOrPlaceholder(ctx),
			"Read container logs for startup errors.",
			"Verify the container exposes the port your Service targets (e.g. port 80 → targetPort).",
		},
		Prevention:  "Align Service targetPort with the port your application listens on; verify with kubectl describe svc.",
		FixCommands: []string{"kubectl get pods -n " + nsOrPlaceholder(ctx), "kubectl logs -n " + nsOrPlaceholder(ctx) + " " + podOrPlaceholder(ctx)},
		Source:      "local",
	}
}

func tunnelBackendFailureResult(ctx DiagnosticContext) AnalysisResult {
	r := applicationUnreachableResult(ctx)
	r.Issue = "Tunnel established but application backend is down"
	r.RootCause = "Port-forward to the service succeeded at the Kubernetes layer, but the backing pod is not serving traffic — commonly CrashLoopBackOff or a crashed process."
	if ctx.PodStatus != "" {
		r.Explanation = "Verified pod status: " + ctx.PodStatus + ". " + r.Explanation
	}
	return r
}

func serviceMisconfigurationResult(ctx DiagnosticContext) AnalysisResult {
	return AnalysisResult{
		Issue:       "Kubernetes service misconfiguration",
		RootCause:   "The expected Service was not found in the cluster for this namespace.",
		Explanation: "Port-forward and deploy commands target a Service that does not exist or uses a different name/namespace.",
		Impact:      "Tunnel and ingress commands cannot route traffic to your application.",
		Resolution: []string{
			"List services: kubectl get svc -A",
			"Confirm the app name matches the Service name in your manifests.",
			"Re-run deployment stages to apply manifests.",
		},
		Prevention:  "Validate Service name and namespace in CI before tunnel stages.",
		FixCommands: []string{"kubectl get svc -A"},
		Source:      "local",
	}
}

func registryUnavailableResult() AnalysisResult {
	return AnalysisResult{
		Issue:       "Local container registry is unavailable",
		RootCause:   "The local Docker registry (127.0.0.1:5001) is not running or not reachable — verified by health check.",
		Explanation: "Kind clusters in this sandbox pull images from the local registry. Generic 'connection refused' on app ports is not treated as a registry failure.",
		Impact:      "Image push and cluster pull steps fail. Deployment cannot use freshly built images.",
		Resolution: []string{
			"Start the registry: docker start local-registry",
			"Or re-run pipeline prep-ci to provision the sandbox.",
			"Verify: curl http://127.0.0.1:5001/v2/_catalog",
		},
		Prevention:  "prep-ci starts and connects the registry before Jenkins builds.",
		FixCommands: []string{"docker ps -f name=local-registry"},
		Source:      "local",
	}
}

func imagePullWithRegistryUpResult() AnalysisResult {
	return AnalysisResult{
		Issue:       "Kubernetes ImagePullBackOff (registry is reachable)",
		RootCause:   "The cluster cannot pull the image even though the registry responded to a health check — likely wrong image name, tag, or image never pushed.",
		Explanation: "Registry availability was verified, so this is not a 'registry down' problem. The image reference in the manifest may be wrong.",
		Impact:      "Pods stay in ImagePullBackOff. The application never starts.",
		Resolution: []string{
			"Verify the image was pushed with the exact name:tag in your Deployment.",
			"Inspect: kubectl describe pod -n <namespace> <pod>",
			"Try pulling manually: docker pull <image>",
		},
		Prevention:  "Pin image tags and push before deploy stages.",
		FixCommands: []string{"kubectl get pods -A", "kubectl describe pod -n <namespace> <pod>"},
		Source:      "local",
	}
}

func nsOrPlaceholder(ctx DiagnosticContext) string {
	if ctx.PodNamespace != "" {
		return ctx.PodNamespace
	}
	if ctx.ServiceNamespace != "" {
		return ctx.ServiceNamespace
	}
	return "<namespace>"
}

func podOrPlaceholder(ctx DiagnosticContext) string {
	if ctx.PodName != "" {
		return ctx.PodName
	}
	return "<pod>"
}

// lowConfidenceHypotheses returns possible causes when evidence is insufficient.
func jenkinsDockerBuildExplanation(ctx DiagnosticContext) string {
	base := "Node.js Docker builds need package.json in the build context to install dependencies before the app is copied in."
	if len(ctx.SkippedStages) > 0 {
		base += " Because the image build failed, Jenkins skipped: " + strings.Join(ctx.SkippedStages, ", ") + "."
	}
	return base
}

func lowConfidenceHypotheses(ctx DiagnosticContext) []string {
	var causes []string
	text := strings.ToLower(corpus(ctx))

	if strings.Contains(text, "connection refused") {
		causes = append(causes, "Application inside the pod is not listening (crash or wrong port)")
		if !ctx.RegistryReachable {
			causes = append(causes, "Local registry unreachable (only if image pull/deploy related)")
		}
		causes = append(causes, "Service or port-forward target misconfigured")
	}
	if strings.Contains(text, "package.json") {
		causes = append(causes, "Missing package.json in Docker build context")
	}
	if len(causes) == 0 {
		causes = append(causes, "Review the failed stage output and re-run with pipeline logs analyze")
	}
	return causes
}
