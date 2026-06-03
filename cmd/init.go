package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"pipeline-cli/core"
	"pipeline-cli/core/summary"
	"pipeline-cli/scaffolding_engine/core/detector"
	"pipeline-cli/scaffolding_engine/core/generator"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes scaffolding for the detected framework",
	Long:  `Detects the project framework in the current directory and generates scaffolding files and a pipeline.yaml starter.`,
	Run: func(cmd *cobra.Command, args []string) {
		core.CommandName = "pipeline init"
		summary.Begin(core.CommandName)
		ok := true
		defer func() { summary.GenerateAndFinish(ok) }()

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			ok = false
			return
		}

		framework, entryPath := detector.DetectFramework(cwd)
		fmt.Printf("Detected framework: %s\n", framework)
		summary.SetMetadata("framework", framework)
		summary.RecordStage("Framework Detection", summary.StageSuccess, framework)

		var aiResult *detector.AIDetectionResult

		if framework == "unknown" {
			fmt.Println("\033[1;36m🤖 Unknown framework detected. Consulting AI for identification...\033[0m")
			result, err := detector.AIDetectFramework(cwd)
			if err != nil {
				fmt.Printf("\033[33m⚠️  AI detection failed: %s\033[0m\n", err.Error())
				fmt.Println("Could not detect or identify the framework.")
				return
			}
			framework = result.Framework
			entryPath = result.EntryPath
			aiResult = &result
		}

		err = generator.GenerateFiles(framework, cwd, entryPath, aiResult)
		if err != nil {
			fmt.Printf("\033[1;31m❌ %s\033[0m\n", err.Error())
			os.Exit(1)
		}
		fmt.Println("Scaffolding generation successful!")

		// Generate pipeline.yaml starter — only if it does not already exist
		yamlPath := filepath.Join(cwd, "pipeline.yaml")
		if _, statErr := os.Stat(yamlPath); os.IsNotExist(statErr) {
			yamlContent := buildPipelineYaml(framework)

			if writeErr := os.WriteFile(yamlPath, []byte(yamlContent), 0644); writeErr != nil {
				fmt.Printf("\033[33m⚠️  Could not write pipeline.yaml: %v\033[0m\n", writeErr)
				summary.AddWarning("pipeline.yaml write failed: " + writeErr.Error())
			} else {
				fmt.Println("\033[1;32m✓\033[0m Generated starter pipeline.yaml")
				summary.RecordInfrastructure("pipeline.yaml", "starter config")
			}
		} else {
			fmt.Println("⚠️  pipeline.yaml already exists — skipping (your customizations are preserved)")
			summary.AddWarning("pipeline.yaml already exists — skipped")
		}
	},
}

func buildPipelineYaml(framework string) string {
	// Shared header
	header := `# pipeline.yaml
# Platform configuration file — commit this to version control.
# All fields are optional. Uncomment and set values to override platform defaults.

version: "1"

app:
  # name: my-service              # Override the app name (default: derived from folder name)
  # port: `

	// Framework-specific config hints
	var portHint, langBlock, healthHint, testHint string

	switch framework {
	case "django", "fastapi":
		portHint = "8000                    # Override the container port (default: 8000)"
		langBlock = `  # python_version: "3.12"        # Override Python version (default: 3.12)`
		healthHint = `  # health_path: "/health"           # Override health check path`
		testHint = `  # test_command: "pytest tests/"     # Override test command`

	case "expressjs", "react":
		if framework == "expressjs" {
			portHint = "3000                    # Override the container port (default: 3000)"
		} else {
			portHint = "8080                    # Override the container port (default: 8080)"
		}

		langBlock = `  # node_version: "22"            # Override Node.js version (default: 22)`
		healthHint = `  # health_path: "/health"           # Override health check path`
		testHint = `  # test_command: "npm test"           # Override test command`

	case "java_springboot":
		portHint = "8080                    # Override the container port (default: 8080)"
		langBlock = `  # java_version: "17"            # Override Java version (default: 17)`
		healthHint = `  # health_path: "/actuator/health"  # Override health check path`
		testHint = `  # test_command: "./mvnw test"        # Override test command`

	default:
		portHint = "8080                    # Override the container port"
		langBlock = ""
		healthHint = `  # health_path: "/health"           # Override health check path`
		testHint = `  # test_command: "your-test-command"  # Override test command`
	}

	// Shared footer
	footer := `
` + healthHint + `
` + testHint + `

# env:
#   - name: ENVIRONMENT
#     value: "production"
#   - name: FEATURE_FLAGS_URL
#     value: "https://flags.internal.yourcompany.com"

# secrets:
#   - name: DATABASE_URL
#     secret_name: my-db-secret
#     secret_key: connection-string

policies:
  mode: opt-out
  # disabled:
  #   - api-versioning
  # config:
  #   feature-flags:
  #     library: "your-flag-library"
`

	return header + portHint + "\n" + langBlock + footer
}

func init() {
	rootCmd.AddCommand(initCmd)
}