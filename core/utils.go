package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// AnalyzeProject returns a map indicating project characteristics
func AnalyzeProject() map[string]bool {
	cwd, _ := os.Getwd()
	
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(cwd, name))
		return err == nil
	}
	
	return map[string]bool{
		"has_docker": exists("Dockerfile"),
		"has_k8s":    exists("k8s") || exists("kubernetes") || exists("kind"),
	}
}

// ExecSilent executes a native OS command silently and returns the error
func ExecSilent(executable string, args ...string) error {
	cmd := exec.Command(executable, args...)
	return cmd.Run()
}

// ExecCommand executes a native OS command without needing a bash wrapper
func ExecCommand(stepName string, ignoreErrors bool, liveOutput bool, executable string, args ...string) {
	fmt.Printf("\n\033[1;36m▶ Running: %s...\033[0m\n", stepName)
	
	cmd := exec.Command(executable, args...)
	
	if liveOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			handleError(err, stepName, ignoreErrors, "")
			return
		}
	} else {
		output, err := cmd.CombinedOutput()
		if err != nil {
			handleError(err, stepName, ignoreErrors, string(output))
			return
		}
	}
	
	fmt.Printf("\033[32m✅ %s completed perfectly!\033[0m\n", stepName)
}

func handleError(err error, stepName string, ignoreErrors bool, output string) {
	if !ignoreErrors {
		fmt.Printf("\033[1;31m❌ Error during %s.\033[0m\n", stepName)
		if output != "" {
			fmt.Print(output)
		}
		os.Exit(1)
	} else {
		fmt.Printf("\033[33m⚠️ %s found issues (Ignored):\033[0m\n", stepName)
		if output != "" {
			fmt.Print(output)
		}
	}
}