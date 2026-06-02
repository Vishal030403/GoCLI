package cmd

import (
	"fmt"
	"io"
	"os"

	"pipeline-cli/core/ai"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "AI-powered log analysis tools",
}

var logsAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze pipeline or Kubernetes logs and get AI-powered fix suggestions",
	Long:  `Reads log content from stdin or a file and uses AI to diagnose failures and suggest fixes.`,
	Run: func(cmd *cobra.Command, args []string) {
		logFile, _ := cmd.Flags().GetString("file")

		var logContent []byte
		var err error

		if logFile != "" {
			logContent, err = os.ReadFile(logFile)
			if err != nil {
				fmt.Printf("\033[1;31m❌ Could not read file: %v\033[0m\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("\033[33mPaste your log output below, then press Ctrl+D when done:\033[0m")
			logContent, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Printf("\033[1;31m❌ Could not read stdin: %v\033[0m\n", err)
				os.Exit(1)
			}
		}

		if len(logContent) == 0 {
			fmt.Println("\033[1;31m❌ No log content provided.\033[0m")
			os.Exit(1)
		}

		fmt.Println("\033[1;36m🤖 Analyzing logs...\033[0m")

		result, err := ai.AnalyzeLogs(string(logContent))
		if err != nil {
			fmt.Printf("\033[1;31m❌ %s\033[0m\n", err.Error())
			os.Exit(1)
		}

		ai.PrintAnalysis(result)
	},
}

func init() {
	logsAnalyzeCmd.Flags().StringP("file", "f", "", "Path to a log file (reads from stdin if not provided)")
	logsCmd.AddCommand(logsAnalyzeCmd)
	rootCmd.AddCommand(logsCmd)
}
