package ai

import (
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// NAMESPACE NAME READY STATUS RESTARTS AGE
	podLineRe = regexp.MustCompile(`^(\S+)\s+(\S+)\s+\d+/\d+\s+(\S+)\s+(\d+)`)
	svcLineRe = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)`)
)

// CollectEvidence runs environment verification commands and appends findings to ctx.
// Safe to call multiple times; only fills empty or unknown fields.
func CollectEvidence(ctx *DiagnosticContext) {
	if ctx == nil {
		return
	}

	if needsKubernetesEvidence(*ctx) {
		collectKubernetesEvidence(ctx)
	}
	if needsDockerEvidence(*ctx) {
		collectDockerEvidence(ctx)
	}
	if needsRegistryEvidence(*ctx) {
		collectRegistryEvidence(ctx)
	}
	if needsJenkinsEvidence(*ctx) {
		collectJenkinsEvidence(ctx)
	}
	inferTunnelEvidence(ctx)
}

func needsKubernetesEvidence(ctx DiagnosticContext) bool {
	text := strings.ToLower(corpus(ctx))
	k8sHints := []string{
		"kubectl", "kubernetes", "pod", "namespace", "crashloop",
		"imagepull", "port-forward", "tunnel", "deployment", "helm",
		"localhost:808", "connection refused",
	}
	for _, h := range k8sHints {
		if strings.Contains(text, h) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(ctx.Command), "tunnel")
}

func needsDockerEvidence(ctx DiagnosticContext) bool {
	text := strings.ToLower(corpus(ctx))
	dockerHints := []string{"docker", "daemon", "container", "image build", "dockerfile"}
	for _, h := range dockerHints {
		if strings.Contains(text, h) {
			return true
		}
	}
	return false
}

func needsRegistryEvidence(ctx DiagnosticContext) bool {
	text := strings.ToLower(corpus(ctx))
	return strings.Contains(text, "registry") ||
		strings.Contains(text, "5001") ||
		strings.Contains(text, "imagepull") ||
		strings.Contains(text, "errimagepull") ||
		strings.Contains(text, "docker push")
}

func needsJenkinsEvidence(ctx DiagnosticContext) bool {
	text := strings.ToLower(corpus(ctx))
	return strings.Contains(text, "jenkins") ||
		strings.Contains(text, "pipeline stage") ||
		len(ctx.SkippedStages) > 0 ||
		ctx.FailedStage != ""
}

func collectKubernetesEvidence(ctx *DiagnosticContext) {
	if _, err := exec.LookPath("kubectl"); err != nil {
		addEvidence(ctx, "kubectl: not installed or not on PATH")
		return
	}

	podsOut, err := exec.Command("kubectl", "get", "pods", "-A", "--no-headers").CombinedOutput()
	if err != nil {
		addEvidence(ctx, "kubectl get pods -A: failed ("+trimOutput(string(podsOut))+")")
	} else {
		parsePodEvidence(ctx, string(podsOut))
		addEvidence(ctx, "kubectl get pods -A: captured")
	}

	svcOut, err := exec.Command("kubectl", "get", "svc", "-A", "--no-headers").CombinedOutput()
	if err != nil {
		addEvidence(ctx, "kubectl get svc -A: failed")
	} else {
		parseServiceEvidence(ctx, string(svcOut))
		addEvidence(ctx, "kubectl get svc -A: captured")
	}
}

func parsePodEvidence(ctx *DiagnosticContext, output string) {
	nsHint := inferNamespace(ctx)
	var worstStatus string
	var worstRestarts int
	var worstPod, worstNS string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := podLineRe.FindStringSubmatch(line)
		if len(m) < 5 {
			continue
		}
		ns, name, status := m[1], m[2], m[3]
		if nsHint != "" && ns != nsHint {
			continue
		}
		restarts := atoi(m[4])
		addEvidence(ctx, "Pod "+ns+"/"+name+": status="+status+", restarts="+strconv.Itoa(restarts))

		if isWorsePodStatus(status, worstStatus) {
			worstStatus = status
			worstRestarts = restarts
			worstPod = name
			worstNS = ns
		}
	}

	if worstStatus != "" {
		ctx.PodStatus = worstStatus
		ctx.PodRestarts = worstRestarts
		ctx.PodName = worstPod
		ctx.PodNamespace = worstNS
		addEvidence(ctx, "Primary pod state: "+worstNS+"/"+worstPod+" → "+worstStatus)
	}
}

func parseServiceEvidence(ctx *DiagnosticContext, output string) {
	nsHint := inferNamespace(ctx)
	targetSvc := inferServiceName(ctx)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := svcLineRe.FindStringSubmatch(line)
		if len(m) < 3 {
			continue
		}
		ns, name := m[1], m[2]
		if nsHint != "" && ns != nsHint {
			continue
		}
		if targetSvc != "" && name != targetSvc {
			continue
		}
		ctx.ServiceExists = true
		ctx.ServiceNamespace = ns
		ctx.ServiceStatus = "exists"
		addEvidence(ctx, "Service "+ns+"/"+name+": exists")
		return
	}

	if targetSvc != "" && nsHint != "" {
		addEvidence(ctx, "Service "+nsHint+"/"+targetSvc+": not found")
		ctx.ServiceExists = false
	}
}

func collectDockerEvidence(ctx *DiagnosticContext) {
	if _, err := exec.LookPath("docker"); err != nil {
		addEvidence(ctx, "docker: not installed")
		ctx.DockerDaemonOK = false
		return
	}

	if err := exec.Command("docker", "info").Run(); err != nil {
		ctx.DockerDaemonOK = false
		addEvidence(ctx, "Docker daemon: unreachable")
		return
	}
	ctx.DockerDaemonOK = true
	addEvidence(ctx, "Docker daemon: healthy")

	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Status}}").Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		addEvidence(ctx, "Running containers: verified")
	}
}

func collectRegistryEvidence(ctx *DiagnosticContext) {
	registryRunning := false
	if _, err := exec.LookPath("docker"); err == nil {
		out, err := exec.Command("docker", "ps", "-f", "name=local-registry", "--format", "{{.Status}}").Output()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			registryRunning = true
			ctx.RegistryStatus = strings.TrimSpace(string(out))
			addEvidence(ctx, "Registry container: "+ctx.RegistryStatus)
		} else {
			addEvidence(ctx, "Registry container (local-registry): not running")
		}
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:5001/v2/_catalog")
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			ctx.RegistryReachable = true
			addEvidence(ctx, "Registry HTTP check (127.0.0.1:5001): reachable")
		} else {
			ctx.RegistryReachable = false
			addEvidence(ctx, "Registry HTTP check: status "+resp.Status)
		}
	} else {
		ctx.RegistryReachable = false
		addEvidence(ctx, "Registry HTTP check (127.0.0.1:5001): unreachable")
	}

	if !registryRunning && !ctx.RegistryReachable {
		ctx.RegistryStatus = "unavailable"
	}
}

func collectJenkinsEvidence(ctx *DiagnosticContext) {
	if _, err := exec.LookPath("docker"); err != nil {
		return
	}
	out, err := exec.Command("docker", "ps", "-f", "name=jenkins", "--format", "{{.Status}}").Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		ctx.JenkinsStatus = strings.TrimSpace(string(out))
		addEvidence(ctx, "Jenkins container: "+ctx.JenkinsStatus)
	} else {
		ctx.JenkinsStatus = "not running"
		addEvidence(ctx, "Jenkins container: not running")
	}
}

func inferTunnelEvidence(ctx *DiagnosticContext) {
	cmd := strings.ToLower(ctx.Command)
	stage := strings.ToLower(ctx.Stage)
	if strings.Contains(cmd, "tunnel") || strings.Contains(stage, "port forward") {
		ctx.PortForwardFailed = ctx.Error != "" || ctx.ExitCode != 0
		errLower := strings.ToLower(ctx.Error + " " + strings.Join(ctx.RecentLogs, " "))
		if strings.Contains(errLower, "forwarding") || strings.Contains(errLower, "port-forward") {
			if !strings.Contains(errLower, "unable to listen") {
				ctx.TunnelEstablished = strings.Contains(errLower, "forwarding from")
			}
		}
		if ctx.PortForwardFailed {
			addEvidence(ctx, "Port forward: failed")
		}
		if ctx.ServiceExists && ctx.PodStatus != "" {
			addEvidence(ctx, "Tunnel target service exists; backend pod state verified")
		}
	}
}

func inferNamespace(ctx *DiagnosticContext) string {
	if ctx.PodNamespace != "" {
		return ctx.PodNamespace
	}
	combined := corpus(*ctx)
	re := regexp.MustCompile(`-n\s+(\S+)`)
	if m := re.FindStringSubmatch(combined); len(m) > 1 {
		return m[1]
	}
	re2 := regexp.MustCompile(`namespace[:\s]+["']?([a-z0-9-]+)`)
	if m := re2.FindStringSubmatch(strings.ToLower(combined)); len(m) > 1 {
		return m[1]
	}
	re3 := regexp.MustCompile(`([a-z0-9-]+-ns)\b`)
	if m := re3.FindStringSubmatch(strings.ToLower(combined)); len(m) > 1 {
		return m[1]
	}
	return ""
}

func inferServiceName(ctx *DiagnosticContext) string {
	combined := corpus(*ctx)
	re := regexp.MustCompile(`svc/([a-z0-9-]+)`)
	if m := re.FindStringSubmatch(combined); len(m) > 1 {
		return m[1]
	}
	return ""
}

func isWorsePodStatus(candidate, current string) bool {
	priority := map[string]int{
		"crashloopbackoff": 5,
		"error":            4,
		"imagepullbackoff": 4,
		"errimagepull":     4,
		"pending":          2,
		"running":          1,
	}
	c := priority[strings.ToLower(candidate)]
	cur := priority[strings.ToLower(current)]
	return c > cur
}

func atoi(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}

func addEvidence(ctx *DiagnosticContext, item string) {
	for _, e := range ctx.Evidence {
		if e == item {
			return
		}
	}
	ctx.Evidence = append(ctx.Evidence, item)
}

func trimOutput(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 120 {
		return s[:120] + "..."
	}
	return s
}
