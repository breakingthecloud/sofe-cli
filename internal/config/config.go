package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIURL        string `yaml:"api_url"`
	APIKey        string `yaml:"api_key"`
	DefaultFormat string `yaml:"default_format"`
	AWSProfile    string `yaml:"aws_profile"`
	PoliciesDir   string `yaml:"policies_dir"`
}

func DefaultConfig() *Config {
	return &Config{
		APIURL:        "http://localhost:8080",
		DefaultFormat: "table",
		PoliciesDir:   "./policies",
	}
}

func Load() *Config {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(filepath.Join(home, ".sofe", "config.yaml"))
	if err != nil {
		return cfg
	}

	_ = yaml.Unmarshal(data, cfg)
	return cfg
}
