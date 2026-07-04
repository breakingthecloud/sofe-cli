package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Mode          string `yaml:"mode"`           // "local" or "cloud"
	APIURL        string `yaml:"api_url"`        // local server URL
	CloudURL      string `yaml:"cloud_url"`      // api.sofe.dev
	APIKey        string `yaml:"api_key"`        // sk_sofe_xxx (for cloud mode)
	DefaultFormat string `yaml:"default_format"`
	AWSProfile    string `yaml:"aws_profile"`
	PoliciesDir   string `yaml:"policies_dir"`
}

func DefaultConfig() *Config {
	return &Config{
		Mode:          "local",
		APIURL:        "http://localhost:8080",
		CloudURL:      "https://api.sofe.dev",
		DefaultFormat: "table",
		PoliciesDir:   "./policies",
	}
}

func Load() *Config {
	cfg := DefaultConfig()

	// Check env vars
	if key := os.Getenv("SOFE_API_KEY"); key != "" {
		cfg.APIKey = key
	}
	if url := os.Getenv("SOFE_CLOUD_URL"); url != "" {
		cfg.CloudURL = url
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(filepath.Join(home, ".sofe", "config.yaml"))
	if err != nil {
		return cfg
	}

	_ = yaml.Unmarshal(data, cfg)

	// Env vars override config file
	if key := os.Getenv("SOFE_API_KEY"); key != "" {
		cfg.APIKey = key
	}

	return cfg
}

func Save(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(home, ".sofe")
	os.MkdirAll(dir, 0700)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0600)
}
