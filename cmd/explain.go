package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:   "explain <eval-id>",
	Short: "Get AI explanation for a finding",
	Long:  "Calls SOFE AI to explain a specific finding from an evaluation.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		evalID := args[0]
		findingIdx, _ := cmd.Flags().GetInt("finding")

		if cfg.APIKey == "" {
			color.Red("❌ No API key configured. Run 'sofe upgrade' first.")
			return
		}

		green := color.New(color.FgGreen, color.Bold)
		cyan := color.New(color.FgCyan)

		// First get the evaluation
		cloudClient := getCloudClient()
		data, err := cloudClient.Get("/evaluations/" + evalID)
		if err != nil {
			color.Red("❌ %s", err)
			return
		}

		var eval struct {
			Findings []struct {
				PolicyName   string  `json:"policy_name"`
				Severity     string  `json:"severity"`
				ResourceID   string  `json:"resource_id"`
				ResourceType string  `json:"resource_type"`
				Message      string  `json:"message"`
			} `json:"findings"`
		}
		json.Unmarshal(data, &eval)

		if findingIdx < 0 || findingIdx >= len(eval.Findings) {
			color.Red("❌ Finding index %d out of range (0-%d)", findingIdx, len(eval.Findings)-1)
			return
		}

		finding := eval.Findings[findingIdx]
		green.Printf("🤖 Explaining finding #%d\n", findingIdx)
		fmt.Printf("   Policy: %s\n", finding.PolicyName)
		fmt.Printf("   Resource: %s\n", finding.ResourceID)
		fmt.Println()

		// Call AI explain
		aiClient := getAIClient()
		explainReq := map[string]interface{}{
			"evaluation_id": evalID,
			"finding_index": findingIdx,
			"finding": map[string]string{
				"policy_name":   finding.PolicyName,
				"severity":      finding.Severity,
				"resource_id":   finding.ResourceID,
				"resource_type": finding.ResourceType,
				"message":       finding.Message,
			},
		}

		respData, err := aiClient.Post("/explain", explainReq)
		if err != nil {
			color.Red("❌ AI error: %s", err)
			return
		}

		var explainResp struct {
			Explanation string `json:"explanation"`
			Model       string `json:"model"`
		}
		json.Unmarshal(respData, &explainResp)

		cyan.Println("─── AI Explanation ───")
		fmt.Println()
		fmt.Println(explainResp.Explanation)
		fmt.Println()
		color.New(color.FgHiBlack).Printf("Model: %s\n", explainResp.Model)
	},
}

func init() {
	explainCmd.Flags().IntP("finding", "f", 0, "Finding index to explain (0-based)")
	rootCmd.AddCommand(explainCmd)
}
