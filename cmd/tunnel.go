package cmd

import (
	"bufio"
	"fmt"
	"io"
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
		summary.BeginTunnel(appName, namespace, "8081", "80")
		summary.RecordInfrastructure("Service", fmt.Sprintf("svc/%s in %s", appName, namespace))

		fmt.Println("\033[1;36m🔄 Patching Kubeconfig for Local Terminal (Native)...\033[0m")
		patchKubeConfig("host.docker.internal", "127.0.0.1")

		fmt.Printf("\033[1;36m🌍 Opening a direct tunnel to '%s'...\033[0m\n", appName)
		fmt.Println("\033[1;32m👉 App will be live at: http://localhost:8081\033[0m")
		fmt.Println("\033[33mPress [Ctrl+C] to close the tunnel when you are done.\n\033[0m")

		interrupted := false
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		c := exec.Command("kubectl", "port-forward", fmt.Sprintf("svc/%s", appName), "8081:80", "-n", namespace)

		stdoutR, stdoutW := io.Pipe()
		stderrR, stderrW := io.Pipe()
		c.Stdout = stdoutW
		c.Stderr = stderrW
		c.Stdin = os.Stdin

		var buf strings.Builder
		go tapPortForwardOutput(stdoutR, &buf)
		go tapPortForwardOutput(stderrR, &buf)

		go func() {
			<-sigChan
			interrupted = true
			if c.Process != nil {
				_ = c.Process.Signal(os.Interrupt)
			}
		}()

		err := c.Start()
		if err != nil {
			_ = stdoutW.Close()
			_ = stderrW.Close()
			failTunnel(appName, err, 1, buf.String())
			return
		}

		err = c.Wait()
		_ = stdoutW.Close()
		_ = stderrW.Close()

		summary.FinalizeTunnel("")

		if interrupted || isInterruptError(err) {
			fmt.Println("\n\033[1;36m🚪 Port-forwarding stopped.\033[0m")
			summary.RecordStage("Port Forward", summary.StageSuccess, "stopped by user (Ctrl+C)")
			summary.FinalizeTunnel("Stopped by user — port-forward session closed cleanly")
			summary.GenerateAndFinish(true)
			return
		}

		if err != nil {
			fmt.Println("\n\033[31m❌ Tunnel disconnected or failed to start.\033[0m")
			failTunnel(appName, err, exitCode(err), buf.String())
			return
		}

		summary.RecordStage("Port Forward", summary.StageSuccess, "tunnel ended")
		summary.FinalizeTunnel("Tunnel session ended")
		summary.GenerateAndFinish(true)
	},
}

func tapPortForwardOutput(r io.Reader, buf *strings.Builder) {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		fmt.Println(line)
		buf.WriteString(line)
		buf.WriteByte('\n')
		if strings.Contains(line, "Handling connection") {
			summary.RecordTunnelRequest()
		}
	}
}

func isInterruptError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "interrupt") ||
		strings.Contains(msg, "killed") ||
		strings.Contains(msg, "signal")
}

func exitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func failTunnel(appName string, err error, code int, output string) {
	summary.MarkFailed()
	summary.RecordStage("Port Forward", summary.StageFailed, err.Error())
	summary.FinalizeTunnel("Tunnel failed: " + err.Error())
	ctx := ai.BuildFailureContext(
		"pipeline tunnel",
		"Port Forward",
		err.Error(),
		code,
		output,
	)
	ai.HandleFailure(ctx)
	summary.GenerateAfterFailure()
	os.Exit(1)
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}
