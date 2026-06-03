package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"pipeline-cli/core"
	"pipeline-cli/core/ai"
	"pipeline-cli/core/summary"

	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Opens a secure port-forward tunnel to your deployed application",
	Run: func(cmd *cobra.Command, args []string) {
		core.CommandName = "pipeline tunnel"
		summary.Begin(core.CommandName)
		cwd, _ := os.Getwd()
		rawName := filepath.Base(cwd)
		appName := strings.ToLower(rawName)
		appName = strings.ReplaceAll(appName, "_", "-")
		appName = strings.ReplaceAll(appName, " ", "-")
		re := regexp.MustCompile(`[^a-z0-9-]`)
		appName = re.ReplaceAllString(appName, "")
		appName = strings.Trim(appName, "-")

		namespace := appName + "-ns"
		summary.RecordInfrastructure("Tunnel", fmt.Sprintf("svc/%s in %s → http://localhost:8081", appName, namespace))
		summary.SetMetadata("app", appName)

		fmt.Println("\033[1;36m🔄 Patching Kubeconfig for Local Terminal (Native)...\033[0m")
		patchKubeConfig("host.docker.internal", "127.0.0.1")

		fmt.Printf("\033[1;36m🌍 Opening a direct tunnel to '%s'...\033[0m\n", appName)
		fmt.Println("\033[1;32m👉 App will be live at: http://localhost:8081\033[0m")
		fmt.Println("\033[33mPress [Ctrl+C] to close the tunnel when you are done.\n\033[0m")

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Println("\n\033[1;36m🚪 Port-forwarding stopped.\033[0m")
			summary.RecordStage("Port Forward", summary.StageSuccess, "stopped by user")
			summary.GenerateAndFinish(true)
			os.Exit(0)
		}()

		c := exec.Command("kubectl", "port-forward", fmt.Sprintf("svc/%s", appName), "8081:80", "-n", namespace)
		stdout, stderr, buf := ai.LiveOutputWriters(30)
		c.Stdout = stdout
		c.Stderr = stderr
		c.Stdin = os.Stdin

		summary.RecordStage("Port Forward", summary.StageSuccess, "tunnel active")

		err := c.Run()
		if err != nil {
			fmt.Println("\n\033[31m❌ Tunnel disconnected or failed to start.\033[0m")
			exitCode := 1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			summary.RecordStage("Port Forward", summary.StageFailed, err.Error())
			ctx := ai.BuildFailureContext(
				"pipeline tunnel",
				"Port Forward",
				err.Error(),
				exitCode,
				buf.Text(),
			)
			ai.HandleFailure(ctx)
			summary.GenerateAfterFailure()
			os.Exit(1)
		}
		summary.GenerateAndFinish(true)
	},
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}