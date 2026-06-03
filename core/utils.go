package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"pipeline-cli/core/ai"
	"pipeline-cli/core/summary"
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

// ExecCommand executes a native OS command without needing a bash wrapper.
// On critical failure, AI analysis runs before exit. Success paths never call AI.
func ExecCommand(stepName string, ignoreErrors bool, liveOutput bool, executable string, args ...string) {
	summary.EnsureSession(activeCommand())
	fmt.Printf("\n\033[1;36m▶ Running: %s...\033[0m\n", stepName)

	cmd := exec.Command(executable, args...)

	if liveOutput {
		stdout, stderr, buf := ai.LiveOutputWriters(30)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err := cmd.Run()
		if err != nil {
			handleError(err, stepName, ignoreErrors, buf.Text())
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
	// ignoreErrors means non-fatal steps still completed successfully — not a warning.
	summary.RecordStage(stepName, summary.StageSuccess, "")
}

func handleError(err error, stepName string, ignoreErrors bool, output string) {
	if !ignoreErrors {
		fmt.Printf("\033[1;31m❌ Error during %s.\033[0m\n", stepName)
		if output != "" {
			fmt.Print(output)
			if output[len(output)-1] != '\n' {
				fmt.Println()
			}
		}

		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}

		summary.EnsureSession(activeCommand())
		summary.MarkFailed()
		summary.RecordStage(stepName, summary.StageFailed, err.Error())
		summary.MarkRemainingSkipped()

		ctx := ai.BuildFailureContext(activeCommand(), stepName, err.Error(), exitCode, output)
		ai.HandleFailure(ctx)
		summary.GenerateAfterFailure()
		os.Exit(1)
	}

	fmt.Printf("\033[33m⚠️ %s found issues (Ignored):\033[0m\n", stepName)
	summary.RecordStage(stepName, summary.StageWarning, err.Error())
	if output != "" {
		fmt.Print(output)
	}
}

func activeCommand() string {
	if CommandName != "" {
		return CommandName
	}
	return "pipeline"
}
