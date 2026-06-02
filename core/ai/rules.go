package ai

import (
	"strings"
)

type knownRule struct {
	patterns []string
	issue    string
	root     string
	explain  string
	impact   string
	fix      []string
	prevent  string
	commands []string
}

var knownFailureRules = []knownRule{
	{
		patterns: []string{"cannot connect to the docker daemon", "error during connect", "docker daemon is not running", "docker daemon unavailable"},
		issue:    "Docker daemon is not running",
		root:     "Docker Desktop or the Docker Engine service is stopped or unreachable.",
		explain:  "The CLI tried to run a container or build command but could not reach the Docker socket.",
		impact:   "Docker builds, registry operations, and Jenkins sandbox steps cannot run. Later pipeline stages are skipped.",
		fix: []string{
			"Start Docker Desktop (Windows/Mac) or run: sudo systemctl start docker (Linux).",
			"Verify connectivity: docker ps",
			"Re-run the failed command after Docker is healthy.",
		},
		prevent:  "Add a preflight check that verifies docker ps succeeds before starting CI stages.",
		commands: []string{"docker ps"},
	},
	{
		patterns: []string{"package.json not found", "npm err! enoent", "could not read package.json"},
		issue:    "package.json not found in Docker build context",
		root:     "The Dockerfile expects package.json (and usually package-lock.json) but they are missing from the build context.",
		explain:  "Node.js Docker builds copy dependency manifests before installing packages. Without them the build fails immediately.",
		impact:   "The Docker image could not be created. Registry push, deployment, and later Jenkins stages are skipped.",
		fix: []string{
			"Verify package.json exists in the project root.",
			"Confirm Jenkins mounts the correct workspace path.",
			"Check the Dockerfile COPY paths match your project layout.",
		},
		prevent:  "Add a pre-build step that fails fast if package.json is absent.",
	},
	{
		patterns: []string{"imagepullbackoff", "errimagepull", "failed to pull image"},
		issue:    "Kubernetes ImagePullBackOff",
		root:     "The cluster cannot pull the container image from the registry.",
		explain:  "The pod spec references an image that is missing, misspelled, or hosted on an unreachable registry.",
		impact:   "Pods stay Pending/CrashLooping. The application never becomes Ready.",
		fix: []string{
			"Verify the image was pushed: docker pull <image>",
			"Check image name and tag in your k8s manifests.",
			"Ensure the cluster can reach the registry (local registry on port 5001 for Kind).",
			"Inspect events: kubectl describe pod -n <namespace> <pod>",
		},
		prevent:  "Pin image tags and validate registry connectivity before deploy stages.",
		commands: []string{"kubectl get pods -A", "kubectl describe pod -n <namespace> <pod>"},
	},
	{
		patterns: []string{"crashloopbackoff", "back-off restarting failed container"},
		issue:    "Kubernetes CrashLoopBackOff",
		root:     "The container starts but exits immediately, causing Kubernetes to restart it repeatedly.",
		explain:  "Common causes include wrong start command, missing env vars, port conflicts, or application startup errors.",
		impact:   "The service never stabilises. Dependent stages and health checks fail.",
		fix: []string{
			"Check container logs: kubectl logs -n <namespace> <pod>",
			"Verify CMD/ENTRYPOINT in the Dockerfile.",
			"Confirm required environment variables and secrets are set.",
		},
		prevent:  "Add liveness/readiness probes and run the container locally before deploying.",
		commands: []string{"kubectl logs -n <namespace> <pod> --previous"},
	},
	{
		patterns: []string{"namespaces \"", "namespace not found", "error from server (notfound)"},
		issue:    "Kubernetes namespace not found",
		root:     "kubectl targeted a namespace that does not exist in the current cluster context.",
		explain:  "Deploy commands reference an app namespace that was never created or was deleted.",
		impact:   "Deployment and tunnel commands fail. The application is not running in the cluster.",
		fix: []string{
			"List namespaces: kubectl get ns",
			"Create the namespace or re-run prep-ci to provision the sandbox.",
			"Verify kubeconfig points to the correct cluster (kind ephemeral-test).",
		},
		prevent:  "Ensure namespace creation is the first step in your deploy stage.",
		commands: []string{"kubectl get ns"},
	},
	{
		patterns: []string{"kubectl: command not found", "executable file not found", "kubectl not found", "lookpath kubectl"},
		issue:    "kubectl is not installed or not on PATH",
		root:     "The kubectl binary is missing from the system PATH.",
		explain:  "Kubernetes operations require kubectl to apply manifests and inspect workloads.",
		impact:   "Cluster validation, deployment, and tunnel commands cannot run.",
		fix: []string{
			"Install kubectl for your OS.",
			"Verify: kubectl version --client",
		},
		prevent:  "Run pipeline preflight checks before starting deploy stages.",
		commands: []string{"kubectl version --client"},
	},
	{
		patterns: []string{"helm: command not found", "helm not found"},
		issue:    "helm is not installed or not on PATH",
		root:     "The helm binary is missing from the system PATH.",
		explain:  "Helm-based deployment stages require the Helm CLI.",
		impact:   "Chart-based deployments cannot proceed.",
		fix: []string{
			"Install Helm: https://helm.sh/docs/intro/install/",
			"Verify: helm version",
		},
		prevent:  "Add helm to preflight dependency checks.",
		commands: []string{"helm version"},
	},
	{
		patterns: []string{"connection refused", "registry unavailable", "dial tcp", "127.0.0.1:5001"},
		issue:    "Local container registry is unavailable",
		root:     "The local Docker registry (127.0.0.1:5001) is not running or not reachable.",
		explain:  "Kind clusters in this sandbox pull images from the local registry bridged to the kind network.",
		impact:   "Image push and cluster pull steps fail. Deployment cannot use freshly built images.",
		fix: []string{
			"Start the registry: docker start local-registry",
			"Or re-run pipeline prep-ci to provision the sandbox.",
			"Verify: curl http://127.0.0.1:5001/v2/_catalog",
		},
		prevent:  "prep-ci starts and connects the registry before Jenkins builds.",
		commands: []string{"docker ps -f name=local-registry"},
	},
	{
		patterns: []string{"kind: command not found", "failed to create cluster"},
		issue:    "Kind cluster creation failed",
		root:     "kind is missing or the cluster could not be created.",
		explain:  "The ephemeral Kubernetes sandbox requires kind to provision a local cluster.",
		impact:   "No cluster is available for deployment or tunnel commands.",
		fix: []string{
			"Install kind and ensure Docker is running.",
			"Delete stale clusters: kind delete cluster --name ephemeral-test",
			"Re-run pipeline prep-ci.",
		},
		prevent:  "Verify kind and Docker in preflight before cluster creation.",
		commands: []string{"kind get clusters"},
	},
}

// matchKnownFailure returns a local diagnosis when error text matches a known pattern.
func matchKnownFailure(ctx FailureContext) (AnalysisResult, bool) {
	text := strings.ToLower(corpus(ctx))

	for _, rule := range knownFailureRules {
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
