package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pipeline-cli/core"
	"pipeline-cli/scaffolding_engine/core/detector"

	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests and linters",
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Run code linters",
	Run: func(cmd *cobra.Command, args []string) { lintCode() },
}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Checks Dockerfile for best practices and optimization",
	Run: func(cmd *cobra.Command, args []string) { lintDocker() },
}

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Validates Kubernetes manifests against official schemas",
	Run: func(cmd *cobra.Command, args []string) { lintK8s() },
}

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Runs a deep security audit on all Infrastructure as Code",
	Run: func(cmd *cobra.Command, args []string) { securityScan() },
}

var unitCmd = &cobra.Command{
	Use:   "unit",
	Short: "Run unit tests",
	Run: func(cmd *cobra.Command, args []string) { unitTests() },
}

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Validates the CI/CD Jenkinsfile for syntax errors",
	Run: func(cmd *cobra.Command, args []string) { lintPipeline() },
}

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all tests",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Starting Full Test Suite...")
		lintCode(); lintDocker(); lintK8s(); securityScan(); unitTests(); lintPipeline()
		fmt.Println("All tests finished.")
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Shift-Left: Runs all local linters, unit tests, and security scans before deployment",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("\033[1;36m🔎 Starting Local Validation (Shift-Left)...\033[0m")
		lintCode(); lintDocker(); lintK8s(); securityScan(); unitTests(); lintPipeline()
		fmt.Println("\n\033[1;32m✅ All local validations passed! Your code is safe and ready for CI deployment.\033[0m")
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(validateCmd)
	testCmd.AddCommand(lintCmd, dockerCmd, k8sCmd, securityCmd, unitCmd, pipelineCmd, allCmd)
}

func lintCode() {
	cwd, _ := os.Getwd()
	framework, _ := detector.DetectFramework(cwd)

	switch framework {
	case "django", "fastapi":
		core.ExecCommand("Python Flake8", true, false, "flake8", ".", "--exclude=env,venv,.git,__pycache__")
	case "expressjs", "react":
		core.ExecCommand("Node.js ESLint", true, false, "npm", "run", "lint")
	case "java_springboot":
		core.ExecCommand("Java Checkstyle", true, false, "./mvnw", "checkstyle:check")
	default:
		fmt.Println("No code linter configured for this framework... Skipping.")
	}
}

func lintDocker() {
	project := core.AnalyzeProject()
	cwd, _ := os.Getwd()
	if project["has_docker"] {
		fmt.Println("Linting Dockerfile...")
		// Cross-platform fix: Mount the directory instead of using the shell '<' redirect
		core.ExecCommand("Hadolint Docker Check", true, false, "docker", "run", "--rm", "-v", fmt.Sprintf("%s:/work", cwd), "-w", "/work", "hadolint/hadolint", "hadolint", "Dockerfile")
	} else {
		fmt.Println("No Dockerfile found. Skipping.")
	}
}

func lintK8s() {
	project := core.AnalyzeProject()
	cwd, _ := os.Getwd()
	if project["has_k8s"] {
		fmt.Println("Validating Kubernetes manifests...")
		
		// Cross-platform fix: Dump to a temp file natively instead of using shell pipes '|'
		tempFile, _ := os.CreateTemp("", "k8s-dump-*.yaml")
		defer os.Remove(tempFile.Name())

		kustomizeCmd := exec.Command("kubectl", "kustomize", filepath.Join(cwd, "k8s/overlays/local"))
		output, _ := kustomizeCmd.Output()
		tempFile.Write(output)
		tempFile.Close()

		core.ExecCommand("Kubeconform K8s Validation", true, false, "docker", "run", "--rm", "-v", fmt.Sprintf("%s:/manifest.yaml", tempFile.Name()), "ghcr.io/yannh/kubeconform:latest", "-strict", "-summary", "/manifest.yaml")
	} else {
		fmt.Println("No Kubernetes manifests found. Skipping.")
	}
}

func securityScan() {
	project := core.AnalyzeProject()
	cwd, _ := os.Getwd()
	if project["has_docker"] || project["has_k8s"] {
		fmt.Println("Running Checkov Security Scan on IaC...")
		core.ExecCommand("Checkov Security Audit", true, false, "docker", "run", "--rm", "-v", fmt.Sprintf("%s:/work", cwd), "bridgecrew/checkov", "-d", "/work", "--framework", "dockerfile", "kubernetes", "github_actions", "--skip-check", "CKV_K8S_14,CKV_K8S_43,CKV2_K8S_6,CKV2_GHA_1,CKV_K8S_40,CKV_K8S_31", "--skip-path", "env", "--skip-path", "venv", "--skip-path", "node_modules", "--skip-path", ".git", "--skip-path", "k8s/overlays", "--quiet", "--compact")
	} else {
		fmt.Println("No infrastructure files found for security scan. Skipping.")
	}
}

func unitTests() {
	cwd, _ := os.Getwd()
	framework, entryPath := detector.DetectFramework(cwd) 

	switch framework {
	case "django":
		core.ExecCommand("Django Unit Tests", true, false, "python3", entryPath, "test")
	case "fastapi":
		core.ExecCommand("FastAPI Pytest", true, false, "pytest")
	case "expressjs", "react":
		core.ExecCommand("Node.js NPM Test", true, false, "npm", "run", "test")
	case "java_springboot":
		core.ExecCommand("Java Spring Boot Tests", true, false, "./mvnw", "test")
	default:
		fmt.Println("No test framework configured... Skipping.")
	}
}

func lintPipeline() {
	cwd, _ := os.Getwd()
	if _, err := os.Stat(cwd + "/Jenkinsfile"); !os.IsNotExist(err) {
		fmt.Println("Validating Jenkinsfile Pipeline...")
		core.ExecCommand("Jenkinsfile Linter", true, false, "docker", "run", "--rm", "-v", fmt.Sprintf("%s:/work", cwd), "-w", "/work", "nvuillam/npm-groovy-lint", "-f", "Jenkinsfile", "--failon", "warning")
	} else {
		fmt.Println("No Jenkinsfile found. Skipping pipeline linting.")
	}
}