package cmd

import (
	"fmt"
	"os"

	"github.com/breakingthecloud/sofe-cli/internal/client"
	"github.com/breakingthecloud/sofe-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfg        *config.Config
	apiClient  *client.Client
	formatFlag string
	Version    string
	Commit     string
	BuildDate  string
)

var rootCmd = &cobra.Command{
	Use:   "sofe",
	Short: "SOFE — Stairway Open FinOps Engine CLI",
	Long:  "Evaluate FinOps policies against live AWS infrastructure via the SOFE API.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg = config.Load()
		apiClient = client.New(cfg.APIURL, cfg.APIKey)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Show welcome banner when sofe is run without subcommand
		if cfg == nil || cfg.APIKey == "" {
			fmt.Println(renderFirstRunBanner())
		} else {
			fmt.Println(renderWelcomeBanner())
		}

		// Non-blocking update check
		if msg := checkForUpdate(); msg != "" {
			fmt.Println()
			fmt.Println(msg)
		}
	},
}

func Execute() {
	rootCmd.Version = Version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "", "Output format: table, json, markdown")
}
