package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

type Evaluation struct {
	ID               string `json:"id"`
	Timestamp        string `json:"timestamp"`
	FindingsCount    int    `json:"findings_count"`
	ResourcesScanned int    `json:"resources_scanned"`
	Trigger          string `json:"trigger"`
}

type HistoryResponse struct {
	Evaluations []Evaluation `json:"evaluations"`
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "List past evaluations (cloud mode)",
	Long:  "Shows your recent evaluations with findings count, resources scanned, and trigger type.",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg.APIKey == "" {
			color.Red("❌ No API key configured. Run 'sofe upgrade' to connect to SOFE Cloud.")
			return
		}

		cloudClient := getCloudClient()
		data, err := cloudClient.Get("/evaluations?limit=15")
		if err != nil {
			color.Red("❌ %s", err)
			return
		}

		var resp HistoryResponse
		json.Unmarshal(data, &resp)

		if len(resp.Evaluations) == 0 {
			fmt.Println("No evaluations yet. Run 'sofe evaluate --cloud' to start.")
			return
		}

		green := color.New(color.FgGreen, color.Bold)
		green.Printf("📊 Evaluation History (%d results)\n\n", len(resp.Evaluations))

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"#", "Date", "Findings", "Resources", "Trigger"})
		table.SetBorder(true)
		table.SetHeaderColor(
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
			tablewriter.Colors{tablewriter.Bold},
			tablewriter.Colors{tablewriter.Bold},
		)

		for i, eval := range resp.Evaluations {
			t, _ := time.Parse(time.RFC3339Nano, eval.Timestamp)
			dateStr := t.Format("Jan 2, 15:04")
			trigger := eval.Trigger
			if trigger == "" {
				trigger = "manual"
			}
			table.Append([]string{
				fmt.Sprintf("%d", i+1),
				dateStr,
				fmt.Sprintf("%d", eval.FindingsCount),
				fmt.Sprintf("%d", eval.ResourcesScanned),
				trigger,
			})
		}
		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
