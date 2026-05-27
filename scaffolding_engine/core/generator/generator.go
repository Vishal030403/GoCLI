package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"pipeline-cli/core/config"
	"pipeline-cli/scaffolding_engine/templates"
)

// GenerateFiles generates the scaffolding files based on the framework.
// It loads pipeline.yaml first, merges user overrides over framework defaults,
// then renders every .tmpl file with the final merged variables.
func GenerateFiles(framework string, projectPath string, entryPath string) error {

	// ── 0. Load user config FIRST — before any defaults are computed ──────────
	userConfig, err := config.LoadConfig(projectPath)
	if err != nil {
		// File is malformed — bubble up so cmd layer can print + exit 1
		return fmt.Errorf("pipeline.yaml is malformed: %v", err)
	}

	// ── 1. Derive DNS-safe app name from the project directory ────────────────
	rawName := filepath.Base(projectPath)
	re := regexp.MustCompile(`[^a-z0-9-]`)
	appName := strings.ToLower(rawName)
	appName = strings.ReplaceAll(appName, "_", "-")
	appName = strings.ReplaceAll(appName, " ", "-")
	appName = re.ReplaceAllString(appName, "")
	appName = strings.Trim(appName, "-")

	// ── 2. Framework defaults ─────────────────────────────────────────────────
	allDefaults := map[string]map[string]interface{}{
		"django": {
			"app_name":       appName,
			"app_port":       8000,
			"python_version": "3.12",
			"run_command":    fmt.Sprintf(`["python", "%s", "runserver", "0.0.0.0:8000"]`, entryPath),
			"health_path":    "/",
			"test_command":   fmt.Sprintf(`python %s test`, entryPath),
		},
		"fastapi": {
			"app_name":       appName,
			"app_port":       8000,
			"python_version": "3.12",
			"run_command":    fmt.Sprintf(`["uvicorn", "%s:app", "--host", "0.0.0.0", "--port", "8000"]`, entryPath),
			"health_path":    "/docs",
			"test_command":   `pytest`,
		},
		"expressjs": {
			"app_name":     appName,
			"app_port":     3000,
			"node_version": "22",
			"run_command":  `["npm", "start"]`,
			"health_path":  "/",
			"test_command": `npm run test`,
		},
		"react": {
			"app_name":     appName,
			"app_port":     8080,
			"node_version": "22",
			"run_command":  `["nginx", "-g", "daemon off;"]`,
			"health_path":  "/",
			"test_command": `npm run test`,
		},
		"java_springboot": {
			"app_name":     appName,
			"app_port":     8080,
			"java_version": "17",
			"run_command":  `["sh", "-c", "java -jar target/*.jar"]`,
			"health_path":  "/actuator/health",
			"test_command": `./mvnw test`,
		},
	}

	frameworkDefaults, ok := allDefaults[framework]
	if !ok {
		frameworkDefaults = map[string]interface{}{"app_name": appName}
	}

	// ── 3. Merge user config over defaults — single merge, used for all files ─
	finalVars := config.MergeWithDefaults(userConfig, frameworkDefaults)

	// ── 4. Walk shared then framework-specific templates ──────────────────────
	dirsToWalk := []string{"shared", framework}

	for _, dir := range dirsToWalk {
		err := fs.WalkDir(templates.Files, dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("templates directory not found for: %s", dir)
			}

			if d.IsDir() {
				return nil
			}

			if strings.HasSuffix(d.Name(), ".tmpl") {
				// Strip the directory prefix to get the output relative path
				relPath := strings.TrimPrefix(path, dir+"/")
				outputRelPath := strings.TrimSuffix(relPath, ".tmpl")
				destPath := filepath.Join(projectPath, outputRelPath)

				if err = os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
					return err
				}

				fileData, err := templates.Files.ReadFile(path)
				if err != nil {
					return err
				}

				// Skip files the developer has already customised
				if _, err := os.Stat(destPath); err == nil {
					fmt.Printf("⚠️  Skipping existing file (already customized): %s\n", outputRelPath)
					return nil
				}

				tmpl, err := template.New(filepath.Base(path)).Parse(string(fileData))
				if err != nil {
					return err
				}

				outFile, err := os.Create(destPath)
				if err != nil {
					return err
				}
				defer outFile.Close()

				if err = tmpl.Execute(outFile, finalVars); err != nil {
					return fmt.Errorf("error executing template %s: %w", path, err)
				}

				fmt.Println("Generated", destPath)
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}