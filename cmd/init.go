package cmd

import (
	"fmt"
	"os"

	"pipeline-cli/scaffolding_engine/core/detector"
	"pipeline-cli/scaffolding_engine/core/generator"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes scaffolding for the detected framework",
	Long:  `Detects the project framework in the current directory and generates scaffolding.`,
	Run: func(cmd *cobra.Command, args []string) {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			return
		}

		// Catch BOTH the framework string and the dynamically found file path
		framework, entryPath := detector.DetectFramework(cwd)
		fmt.Printf("Detected framework: %s\n", framework)

		if framework != "unknown" {
			// Pass entryPath to the generator!
			err = generator.GenerateFiles(framework, cwd, entryPath)
			if err != nil {
				fmt.Println("Error generating files:", err)
			} else {
				fmt.Println("Scaffolding generation successful!")
			}
		} else {
			fmt.Println("Could not detect a supported framework.")
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
