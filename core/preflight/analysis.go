package preflight

import (
	"fmt"
	"os"

	"pipeline-cli/core/ai"
)

const prepCICommand = "pipeline prep-ci"

// exitWithAnalysis prints a fatal preflight error, runs local/Gemini analysis, and exits.
func exitWithAnalysis(stage, message string) {
	fmt.Printf("\033[1;31m❌ %s\033[0m\n", message)
	ctx := ai.BuildFailureContext(prepCICommand, stage, message, 1, message)
	ai.HandleFailure(ctx)
	os.Exit(1)
}
