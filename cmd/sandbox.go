package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"pipeline-cli/core"
	"pipeline-cli/core/preflight" // Keep this only for CheckDependencies and EnsureDockerRunning

	"github.com/spf13/cobra"
)

var prepCiCmd = &cobra.Command{
	Use:   "prep-ci",
	Short: "Spins up an empty ephemeral cluster, registry, and Jenkins sandbox",
	Run: func(cmd *cobra.Command, args []string) {
		
		// 1. RUN PREFLIGHT (Only checks deps and starts docker now)
		preflight.RunSetupChecks()
		
		clusterName := "ephemeral-test"

		// 2. HARDCODED REGISTRY (Port 5001)
		if !isRegistryRunning() {
			fmt.Println("\n\033[33m⚠️ Local registry not running. Waking it up...\033[0m")
			err := core.ExecSilent("docker", "start", "local-registry")
			if err != nil {
				core.ExecCommand("Starting Registry", true, true, "docker", "run", "-d", "--restart=always", "-p", "5001:5000", "--name", "local-registry", "-v", "local-registry-data:/var/lib/registry", "registry:2")
			}
		}

		fmt.Println("\n\033[1;36m🏗️ Building Kubernetes Sandbox & CI/CD Pipeline...\033[0m")

		// 3. KIND CONFIG (Forcing IPv4 127.0.0.1 to bypass the [::1] timeout)
		kindConfig := `
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."127.0.0.1:5001"]
    endpoint = ["http://local-registry:5000"]
`
		tempFile, _ := os.CreateTemp("", "kind-config-*.yaml")
		defer os.Remove(tempFile.Name())
		tempFile.WriteString(kindConfig)
		tempFile.Close()

		core.ExecCommand("Creating empty Kind cluster", false, true, "kind", "create", "cluster", "--name", clusterName, "--config", tempFile.Name(), "--image", "kindest/node:v1.30.0")
		core.ExecCommand("Bridging Registry and Kind Networks", true, false, "docker", "network", "connect", "kind", "local-registry")

		fmt.Println("\033[1;36m🔄 Patching Kubeconfig for Jenkins (Native)...\033[0m")
		patchKubeConfig("127.0.0.1", "host.docker.internal")

		cascYaml := `
jenkins:
  securityRealm:
    local:
      allowsSignup: false
      users:
       - id: "admin"
         password: "admin"
  authorizationStrategy:
    loggedInUsersCanDoAnything:
      allowAnonymousRead: false
`
		cascFile, _ := os.CreateTemp("", "casc-*.yaml")
		defer os.Remove(cascFile.Name())
		cascFile.WriteString(cascYaml)
		cascFile.Close()

		if !isJenkinsRunning() {
			fmt.Println("\033[1;36m🚀 Launching Jenkins Server (Automated Setup)...\033[0m")
			
			err := core.ExecSilent("docker", "start", jenkinsName)

			if err != nil {
				homeDir, _ := os.UserHomeDir()
				
				// 4. HARDCODED JENKINS (Port 8080)
				core.ExecCommand("Booting Jenkins Container", true, false, "docker", "run", "-d", "--restart=always", "-p", "8080:8080", "-p", "50000:50000", "--name", jenkinsName, "-u", "root", "-e", fmt.Sprintf("HOST_HOME=%s", homeDir), "-e", `JAVA_OPTS="-Djenkins.install.runSetupWizard=false"`, "-e", "CASC_JENKINS_CONFIG=/var/jenkins_home/casc.yaml", "-v", "local-jenkins-data:/var/jenkins_home", "-v", "/var/run/docker.sock:/var/run/docker.sock", "jenkins/jenkins:lts")

				core.ExecCommand("Installing Docker CLI inside Jenkins (Takes ~2 min)", false, false, "docker", "exec", "-u", "root", jenkinsName, "bash", "-c", "apt-get update && apt-get install -y docker.io")
				core.ExecCommand("Installing Kustomize inside Jenkins", false, false, "docker", "exec", "-u", "root", jenkinsName, "bash", "-c", `curl -sSL "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash && mv kustomize /usr/local/bin/`)
				core.ExecCommand("Installing Jenkins Plugins (Takes ~2 min)", false, false, "docker", "exec", jenkinsName, "jenkins-plugin-cli", "--plugins", "git", "workflow-aggregator", "docker-workflow", "configuration-as-code", "ws-cleanup")
				core.ExecCommand("Injecting JCasC Configuration", true, false, "docker", "cp", cascFile.Name(), fmt.Sprintf("%s:/var/jenkins_home/casc.yaml", jenkinsName))
				core.ExecCommand("Applying configurations", false, true, "docker", "restart", jenkinsName)

				fmt.Println("\033[33m⏳ Waiting for Jenkins to fully boot...\033[0m")
				core.ExecCommand("Checking Jenkins API readiness", false, true, "docker", "exec", jenkinsName, "bash", "-c", `until curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/login | grep -q "200"; do sleep 3; done`)

				// Auto Job Creation Logic
				cwd, _ := os.Getwd()
				rawName := filepath.Base(cwd)
				appName := strings.ToLower(rawName)
				appName = strings.ReplaceAll(appName, "_", "-")
				appName = strings.ReplaceAll(appName, " ", "-")

				fmt.Println("\n\033[1;36m🤖 Automating Jenkins Pipeline Creation...\033[0m")
				
				gitCmd := exec.Command("git", "config", "--get", "remote.origin.url")
				gitOut, gitErr := gitCmd.Output()
				repoURL := strings.TrimSpace(string(gitOut))

				if gitErr != nil || repoURL == "" {
					fmt.Println("\033[33m⚠️ Could not detect Git remote URL. Job auto-creation skipped.\033[0m")
				} else {
					fmt.Printf("🔗 Linking Job '%s' to Repository: %s\n", appName, repoURL)

					jobXML := fmt.Sprintf(`<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job">
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsScmFlowDefinition" plugin="workflow-cps">
    <scm class="hudson.plugins.git.GitSCM" plugin="git">
      <userRemoteConfigs>
        <hudson.plugins.git.UserRemoteConfig>
          <url>%s</url>
        </hudson.plugins.git.UserRemoteConfig>
      </userRemoteConfigs>
      <branches>
        <hudson.plugins.git.BranchSpec>
          <name>*/main</name>
        </hudson.plugins.git.BranchSpec>
      </branches>
    </scm>
    <scriptPath>Jenkinsfile</scriptPath>
    <lightweight>true</lightweight>
  </definition>
</flow-definition>`, repoURL)

					xmlFile, _ := os.CreateTemp("", "job-*.xml")
					xmlFile.WriteString(jobXML)
					xmlFile.Close()
					defer os.Remove(xmlFile.Name())

					apiScript := fmt.Sprintf(`#!/bin/bash
CRUMB=$(curl -s -c /tmp/cookies.txt -u admin:admin "http://localhost:8080/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)")
curl -s -X POST "http://localhost:8080/createItem?name=%s" -u admin:admin -b /tmp/cookies.txt -H "$CRUMB" -H "Content-Type:text/xml" -d @/tmp/job.xml
curl -s -X POST "http://localhost:8080/job/%s/build" -u admin:admin -b /tmp/cookies.txt -H "$CRUMB"
`, appName, appName)

					scriptFile, _ := os.CreateTemp("", "setup-*.sh")
					scriptFile.WriteString(apiScript)
					scriptFile.Close()
					defer os.Remove(scriptFile.Name())

					core.ExecCommand("Injecting XML Blueprint", true, false, "docker", "cp", xmlFile.Name(), fmt.Sprintf("%s:/tmp/job.xml", jenkinsName))
					core.ExecCommand("Injecting API Script", true, false, "docker", "cp", scriptFile.Name(), fmt.Sprintf("%s:/tmp/setup.sh", jenkinsName))
					core.ExecCommand("Executing Jenkins API Configurator", true, false, "docker", "exec", jenkinsName, "bash", "/tmp/setup.sh")

					fmt.Println("\033[1;36m🎯 Your pipeline job is configured and building!\033[0m")
				}
			} else {
				fmt.Printf("✅ Reconnected to existing '%s' container.\n", jenkinsName)
			}
		} else {
			fmt.Printf("\033[1;32m✅ Jenkins '%s' is active.\033[0m\n", jenkinsName)
		}

		fmt.Println("\n\033[1;32m✅ CI/CD Sandbox is LIVE and Ready!\033[0m")
		
		fmt.Printf("\033[33m👉 Jenkins UI: http://localhost:8080\033[0m\n")
		fmt.Printf("\033[33m👉 Docker Push API: 127.0.0.1:5001\033[0m\n")
		fmt.Println("\033[33m👉 Credentials: admin / admin\033[0m\n")
	},
}

var destroyCiCmd = &cobra.Command{
	Use:   "destroy-ci",
	Short: "Completely destroys the local CI/CD sandbox",
	Run: func(cmd *cobra.Command, args []string) {
		clusterName := "ephemeral-test"

		fmt.Println("\033[1;31m💥 Commencing total teardown...\033[0m")
		
		core.ExecCommand("Nuking containers", true, false, "docker", "rm", "-f", jenkinsName, "local-registry", "jenkins-sandbox")
		core.ExecCommand("Wiping persistent data", true, false, "docker", "volume", "rm", "local-jenkins-data", "local-registry-data")
		core.ExecCommand("Destroying Kind cluster", true, true, "kind", "delete", "cluster", "--name", clusterName)

		fmt.Println("\033[1;36m🔄 Restoring network context (Native)...\033[0m")
		patchKubeConfig("host.docker.internal", "127.0.0.1")

		fmt.Println("\n\033[1;32m🧹 Clean slate! Everything destroyed safely.\033[0m\n")
	},
}

func init() {
	rootCmd.AddCommand(prepCiCmd)
	rootCmd.AddCommand(destroyCiCmd)
}

const jenkinsName = "local-jenkins"

func isJenkinsRunning() bool {
	cmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=%s", jenkinsName))
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func isRegistryRunning() bool {
	cmd := exec.Command("docker", "ps", "-q", "-f", "name=local-registry")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func patchKubeConfig(oldStr string, newStr string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("\033[31m❌ Could not find home directory to patch kubeconfig\033[0m")
		return
	}
	kubeConfigPath := filepath.Join(homeDir, ".kube", "config")
	input, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		fmt.Println("\033[31m❌ Could not read ~/.kube/config\033[0m")
		return
	}
	output := strings.ReplaceAll(string(input), oldStr, newStr)
	err = os.WriteFile(kubeConfigPath, []byte(output), 0644)
	if err != nil {
		fmt.Println("\033[31m❌ Could not write to ~/.kube/config\033[0m")
		return
	}
}