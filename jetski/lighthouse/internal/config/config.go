package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port         int    `yaml:"port"`
	SecretKey    string `yaml:"secret_key"`
	DatabasePath string `yaml:"database_path"`
}

func Load() (*Config, error) {
	// Load default config from YAML
	config := &Config{
		Port:         8080,
		SecretKey:    "change-me-in-production",
		DatabasePath: "./data/lighthouse.db",
	}

	// Read YAML file
	data, err := os.ReadFile("configs/config.yaml")
	if err == nil {
		yaml.Unmarshal(data, config)
	}

	// Override with environment variables if set
	if port := os.Getenv("LIGHTHOUSE_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}
	if secretKey := os.Getenv("LIGHTHOUSE_SECRET_KEY"); secretKey != "" {
		config.SecretKey = secretKey
	}
	if dbPath := os.Getenv("LIGHTHOUSE_DATABASE_PATH"); dbPath != "" {
		config.DatabasePath = dbPath
	}

	return config, nil
}
