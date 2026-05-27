package config

// MergeWithDefaults merges a loaded PipelineConfig over a map of framework defaults.
// User values always win; unset fields fall back to framework defaults.
func MergeWithDefaults(userConfig PipelineConfig, defaults map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range defaults {
		merged[k] = v
	}

	if userConfig.App.Name != "" {
		merged["app_name"] = userConfig.App.Name
	}
	if userConfig.App.Port != 0 {
		merged["app_port"] = userConfig.App.Port
	}
	if userConfig.App.PythonVersion != "" {
		merged["python_version"] = userConfig.App.PythonVersion
	}
	if userConfig.App.NodeVersion != "" {
		merged["node_version"] = userConfig.App.NodeVersion
	}
	if userConfig.App.JavaVersion != "" {
		merged["java_version"] = userConfig.App.JavaVersion
	}
	if userConfig.App.HealthPath != "" {
		merged["health_path"] = userConfig.App.HealthPath
	}
	if userConfig.App.TestCommand != "" {
		merged["test_command"] = userConfig.App.TestCommand
	}

	merged["env_vars"] = userConfig.App.Env
	merged["secrets"] = userConfig.App.Secrets
	return merged
}