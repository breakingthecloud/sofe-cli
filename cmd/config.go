package cmd

import (
	"fmt"

	"github.com/breakingthecloud/sofe-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage sofe-cli configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a config value (api-key, mode, cloud-url, profile)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		switch key {
		case "api-key":
			cfg.APIKey = value
		case "mode":
			if value != "local" && value != "cloud" {
				return fmt.Errorf("mode must be 'local' or 'cloud'")
			}
			cfg.Mode = value
		case "cloud-url":
			cfg.CloudURL = value
		case "profile":
			cfg.AWSProfile = value
		case "format":
			cfg.DefaultFormat = value
		default:
			return fmt.Errorf("unknown config key: %s (available: api-key, mode, cloud-url, profile, format)", key)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✅ Set %s = %s\n", key, value)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("mode:       %s\n", cfg.Mode)
		fmt.Printf("api_url:    %s\n", cfg.APIURL)
		fmt.Printf("cloud_url:  %s\n", cfg.CloudURL)
		if cfg.APIKey != "" {
			fmt.Printf("api_key:    %s...%s\n", cfg.APIKey[:8], cfg.APIKey[len(cfg.APIKey)-4:])
		} else {
			fmt.Printf("api_key:    (not set)\n")
		}
		fmt.Printf("profile:    %s\n", cfg.AWSProfile)
		fmt.Printf("format:     %s\n", cfg.DefaultFormat)
		fmt.Printf("policies:   %s\n", cfg.PoliciesDir)
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
