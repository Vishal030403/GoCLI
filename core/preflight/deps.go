package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// CheckDependencies ensures required binaries are in the system PATH, and tries to install them if not.
func CheckDependencies() error {
	deps := []string{"docker", "kind", "kubectl"}

	for _, dep := range deps {
		_, err := exec.LookPath(dep)
		if err != nil {
			fmt.Printf("\033[33m⚠️ Missing dependency: %s. Attempting auto-install...\033[0m\n", dep)
			
			installErr := installDependency(dep)
			if installErr != nil {
				return fmt.Errorf("could not auto-install %s. Please install it manually (%v)", dep, installErr)
			}
			fmt.Printf("\033[1;32m✅ Successfully installed %s!\033[0m\n", dep)
		}
	}
	return nil
}

// installDependency uses OS-specific package managers to download missing tools
func installDependency(dep string) error {
	// Docker requires system-level permissions and restarts, so we force the user to install it manually.
	if dep == "docker" {

	switch runtime.GOOS {

	case "darwin":

		// Validate Homebrew exists
		if _, err := exec.LookPath("brew"); err != nil {

			return fmt.Errorf(
				"Homebrew is required.\nInstall from: https://brew.sh",
			)
		}

		cmd := exec.Command(
			"brew",
			"install",
			"--cask",
			"docker",
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()

	case "linux":

		cmd := exec.Command(
			"bash",
			"-c",
			"curl -fsSL https://get.docker.com | sh",
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()

	case "windows":

		return fmt.Errorf(
			"automatic Docker installation on Windows is not yet supported safely",
		)
	}
}

	switch runtime.GOOS {
	case "darwin": // Mac
		return exec.Command("brew", "install", dep).Run()
		
	case "windows": // Windows Native / PowerShell
		if dep == "kind" {
			return exec.Command("winget", "install", "Kubernetes.kind").Run()
		}
		if dep == "kubectl" {
			return exec.Command("winget", "install", "Kubernetes.kubectl").Run()
		}
		
	case "linux": // WSL / Ubuntu
		// The safest cross-platform way to install go binaries on Linux without fighting apt/snap
		if dep == "kind" {
			return exec.Command("bash", "-c", "curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64 && chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind").Run()
		}
		if dep == "kubectl" {
			return exec.Command("bash", "-c", `curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && chmod +x ./kubectl && sudo mv ./kubectl /usr/local/bin/kubectl`).Run()
		}
	}

	return fmt.Errorf("unsupported OS for auto-install")
}