package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads pipeline.yaml from the given directory.
// Returns empty config (no error) if the file does not exist.
func LoadConfig(dir string) (PipelineConfig, error) {
	var cfg PipelineConfig
	configPath := filepath.Join(dir, "pipeline.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("failed to read pipeline.yaml: %v", err)
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("pipeline.yaml is malformed. Check syntax: %v", err)
	}
	return cfg, nil
}