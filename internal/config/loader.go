package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ConfigFileName = "agentbox.yaml"

// Load reads configuration from the given project directory
func Load(projectDir string) (*Config, error) {
	configPath := filepath.Join(projectDir, ConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes configuration to the given project directory
func Save(projectDir string, cfg *Config) error {
	configPath := filepath.Join(projectDir, ConfigFileName)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// Exists checks if a configuration file exists in the given directory
func Exists(projectDir string) bool {
	configPath := filepath.Join(projectDir, ConfigFileName)
	_, err := os.Stat(configPath)
	return err == nil
}
