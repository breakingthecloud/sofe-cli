package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current account status (tier, evals, AI, accounts)",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg.APIKey == "" {
			color.Yellow("Mode: self-hosted (local only)")
			fmt.Println("  No API key configured.")
			fmt.Println("  Run 'sofe upgrade' to connect to SOFE Cloud.")
			return
		}

		green := color.New(color.FgGreen, color.Bold)
		green.Println("┌─────────────────────────────────────────┐")
		green.Println("│ SOFE Status                              │")
		green.Println("├─────────────────────────────────────────┤")

		cloudClient := getCloudClient()

		// Get latest evaluation
		data, err := cloudClient.Get("/evaluations?limit=1")
		if err != nil {
			fmt.Printf("│ ❌ API error: %s\n", err)
			green.Println("└─────────────────────────────────────────┘")
			return
		}

		var histResp HistoryResponse
		json.Unmarshal(data, &histResp)

		// Get AI usage
		aiClient := getAIClient()
		aiData, _ := aiClient.Get("/usage")
		var aiUsage struct {
			CallsToday int    `json:"calls_today"`
			Date       string `json:"date"`
		}
		if aiData != nil {
			json.Unmarshal(aiData, &aiUsage)
		}

		// Display
		fmt.Printf("│ Mode:        cloud                      │\n")
		fmt.Printf("│ API Key:     %s...%s │\n", cfg.APIKey[:12], cfg.APIKey[len(cfg.APIKey)-4:])
		fmt.Printf("│ Cloud URL:   %s         │\n", cfg.CloudURL)
		if len(histResp.Evaluations) > 0 {
			last := histResp.Evaluations[0]
			fmt.Printf("│ Last eval:   %d findings (%s)    │\n", last.FindingsCount, last.Trigger)
		} else {
			fmt.Printf("│ Last eval:   none                       │\n")
		}
		fmt.Printf("│ AI today:    %d calls                    │\n", aiUsage.CallsToday)
		green.Println("└─────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("Tip: 'sofe history' for full evaluation list")
		fmt.Println("     'sofe evaluate --cloud' for a new scan")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
