package policies

import (
	"strings"

	"pipeline-cli/core/policy"
)

// LoggingStandard checks for bare logging calls that bypass proper structured loggers.
type LoggingStandard struct{}

func (p *LoggingStandard) Name() string        { return "logging-standard" }
func (p *LoggingStandard) DisplayName() string { return "Logging Standard" }
func (p *LoggingStandard) Category() string    { return "standards" }
func (p *LoggingStandard) Severity() string    { return "warning" }
func (p *LoggingStandard) Description() string {
	return "Finds bare logging calls that bypass structured logging: print() in Python, console.log/error/warn in JS/TS, and System.out.println in Java. Test files are excluded."
}

func (p *LoggingStandard) Run(projectPath string, _ map[string]map[string]interface{}) policy.PolicyResult {
	result := policy.PolicyResult{
		PolicyName: p.Name(),
		Severity:   p.Severity(),
		Passed:     true,
	}

	framework := detectFramework(projectPath)
	var findings []policy.Finding

	walkSourceFiles(projectPath, func(path string) {
		if isTestFile(path) {
			return
		}

		switch framework {
		case "django", "fastapi", "python":
			if !strings.HasSuffix(path, ".py") {
				return
			}
			findings = append(findings, scanForPattern(projectPath, path, "print(")...)

		case "expressjs", "react", "node":
			if !strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".ts") &&
				!strings.HasSuffix(path, ".jsx") && !strings.HasSuffix(path, ".tsx") {
				return
			}
			findings = append(findings, scanForPattern(projectPath, path, "console.log(")...)
			findings = append(findings, scanForPattern(projectPath, path, "console.error(")...)
			findings = append(findings, scanForPattern(projectPath, path, "console.warn(")...)

		case "java_springboot":
			if !strings.HasSuffix(path, ".java") {
				return
			}
			findings = append(findings, scanForPattern(projectPath, path, "System.out.println(")...)
		}
	})

	if len(findings) > 0 {
		result.Passed = false
		result.Findings = findings
	}
	return result
}

// scanForPattern returns a Finding for every line in path that contains pattern.
func scanForPattern(projectPath, path, pattern string) []policy.Finding {
	lines := readLines(path)
	if lines == nil {
		return nil
	}
	rel, _ := relPath(projectPath, path)
	var findings []policy.Finding
	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.Contains(line, pattern) {
			findings = append(findings, policy.Finding{
				File:   rel,
				Line:   lineNum + 1,
				Detail: "bare logging call: " + strings.TrimSpace(pattern),
			})
		}
	}
	return findings
}

// relPath is a thin wrapper to avoid importing filepath in multiple places.
func relPath(base, target string) (string, error) {
	import_path_helper := target
	if len(base) > 0 && strings.HasPrefix(target, base) {
		import_path_helper = strings.TrimPrefix(target, base)
		import_path_helper = strings.TrimPrefix(import_path_helper, "/")
		import_path_helper = strings.TrimPrefix(import_path_helper, "\\")
	}
	return import_path_helper, nil
}
