package preflight

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const dockerVersionTimeout = 30 * time.Second

func EnsureDockerRunning() error {

	fmt.Println("🔍 Validating Docker daemon...")

	// ------------------------------------------------------------
	// 1. Check if Docker is installed
	// ------------------------------------------------------------

	if _, err := exec.LookPath("docker"); err != nil {

		fmt.Println("⚠️ Docker not installed — attempting installation...")

		switch runtime.GOOS {

		case "darwin":

			// Validate Homebrew
			if _, err := exec.LookPath("brew"); err != nil {

				return errors.New(
					"Homebrew is required.\n" +
						"Install from: https://brew.sh",
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

			if err := cmd.Run(); err != nil {

				return fmt.Errorf(
					"docker installation failed: %w",
					err,
				)
			}

			fmt.Println("✅ Docker Desktop installed successfully")

		case "linux":

			cmd := exec.Command(
				"sh",
				"-c",
				"curl -fsSL https://get.docker.com | sh",
			)

			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {

				return fmt.Errorf(
					"docker installation failed: %w",
					err,
				)
			}

			fmt.Println("✅ Docker installed successfully")

		case "windows":

			return errors.New(
				"automatic Docker installation on Windows is not yet supported safely.\n" +
					"Please install Docker Desktop manually from:\n" +
					"https://www.docker.com/products/docker-desktop",
			)

		default:

			return errors.New("unsupported operating system")
		}
	}

	// ------------------------------------------------------------
	// 2. Validate Docker daemon
	// ------------------------------------------------------------

	version, err := runDockerVersion()

	if err == nil {

		fmt.Println("✅ Docker daemon healthy")
		fmt.Println("🐳 Docker Engine v" + version)

		return nil
	}

	fmt.Println("⚠️ Docker daemon unreachable — attempting recovery...")

	// ------------------------------------------------------------
	// 3. OS-specific daemon recovery
	// ------------------------------------------------------------

	switch runtime.GOOS {

	case "darwin":

		fmt.Println("🚀 Launching Docker Desktop...")

		if err := exec.Command(
			"open",
			"-a",
			"Docker",
		).Start(); err != nil {

			return fmt.Errorf(
				"failed to launch Docker Desktop: %w",
				err,
			)
		}

	case "windows":

		fmt.Println("🚀 Launching Docker Desktop...")

		if err := exec.Command(
			"powershell",
			"-Command",
			"Start-Process Docker Desktop",
		).Start(); err != nil {

			return fmt.Errorf(
				"failed to launch Docker Desktop: %w",
				err,
			)
		}

	case "linux":

		if isWSL() {

			fmt.Println("🪟 Detected WSL environment")

			exec.Command(
				"cmd.exe",
				"/c",
				"start",
				"",
				"C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe",
			).Start()

		} else {

			return errors.New(
				"docker daemon unavailable.\n\n" +
					"Run:\n" +
					"sudo systemctl start docker\n" +
					"sudo usermod -aG docker $USER",
			)
		}
	}

	// ------------------------------------------------------------
	// 4. Wait for Docker daemon readiness
	// ------------------------------------------------------------

	timeout := time.After(180 * time.Second)

	ticker := time.NewTicker(2 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-timeout:

			return errors.New(
				"docker daemon startup timeout exceeded",
			)

		case <-ticker.C:

			version, err := runDockerVersion()

			if err == nil {

				fmt.Println("✅ Docker daemon recovered")
				fmt.Println("🐳 Docker Engine v" + version)

				return nil
			}

			fmt.Println("⏳ Waiting for Docker daemon...")
		}
	}
}

// ------------------------------------------------------------
// Docker version validation
// ------------------------------------------------------------

func runDockerVersion() (string, error) {

	ctx, cancel := context.WithTimeout(
		context.Background(),
		dockerVersionTimeout,
	)

	defer cancel()

	out, err := exec.CommandContext(
		ctx,
		"docker",
		"version",
		"--format",
		"{{.Server.Version}}",
	).Output()

	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(out))

	if version == "" {
		return "unknown", nil
	}

	return version, nil
}

// ------------------------------------------------------------
// WSL detection
// ------------------------------------------------------------

func isWSL() bool {

	data, err := os.ReadFile("/proc/version")

	if err != nil {
		return false
	}

	return strings.Contains(
		strings.ToLower(string(data)),
		"microsoft",
	)
}