package cmd

import (
	"fmt"
	"os"

	"pipeline-cli/core/summary"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "My DevOps Pipeline CLI",
	Long:  `A local CI/CD pipeline and observability CLI tool.`,
	PersistentPostRun: func(c *cobra.Command, _ []string) {
		switch c.Name() {
		case "init", "prep-ci", "tunnel", "destroy-ci":
			summary.FlushPending()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
