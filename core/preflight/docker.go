package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func EnsureDockerRunning() error {
	// 0. NEW: Check if Docker is even installed!
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Docker is missing! \033[33m👉 Please download and install Docker Desktop from: https://www.docker.com/products/docker-desktop\033[0m")
	}

	// 1. Check if Docker daemon responds
	err = exec.Command("docker", "info").Run()
	if err == nil {
		return nil // It's already running!
	}

	fmt.Println("\033[33m⚠️ Docker is not running. Attempting auto-start...\033[0m")

	switch runtime.GOOS {
	case "darwin": // Mac
		exec.Command("open", "-a", "Docker").Run()
	case "windows": // Native Windows Command Prompt
		exec.Command("powershell", "-Command", "Start-Process 'C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe'").Run()
	case "linux":
		// 2. Check if we are in WSL or Native Linux
		if isWSL() {
			// We are in WSL! Reach out to Windows and start Docker Desktop
			exec.Command("cmd.exe", "/c", "start", "", "C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe").Run()
		} else {
			// Native Linux (Ubuntu Server, etc.)
			// We don't want to freeze the CLI with a sudo password prompt.
			return fmt.Errorf("Docker daemon is down. Please run 'sudo systemctl start docker' manually")
		}
	}

	// 3. Poll until it wakes up (max 30 seconds)
	for i := 0; i < 60; i++ {
		if exec.Command("docker", "info").Run() == nil {
			fmt.Println("\033[1;32m✅ Docker started successfully.\033[0m")
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("failed to start Docker automatically. Please start Docker Desktop manually")
}

// isWSL reads the Linux kernel version to see if Microsoft compiled it
func isWSL() bool {
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(b)), "microsoft")
}