package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Top findings ranked by frequency across evaluations",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg.APIKey == "" {
			color.Yellow("No API key configured. Run 'sofe upgrade' or set key.")
			return
		}

		cloudClient := getCloudClient()

		// Get latest evaluation (aggregation from one eval is still useful)
		data, err := cloudClient.Get("/evaluations?limit=1")
		if err != nil {
			color.Red("Error: %s", err)
			return
		}
		var hist HistoryResponse
		json.Unmarshal(data, &hist)
		if len(hist.Evaluations) == 0 {
			fmt.Println("No evaluations found.")
			return
		}

		// Get findings from latest
		evalData, err := cloudClient.Get("/evaluations/" + hist.Evaluations[0].ID)
		if err != nil {
			color.Red("Error: %s", err)
			return
		}

		var evalResp struct {
			Findings []struct {
				PolicyName   string `json:"policy_name"`
				Severity     string `json:"severity"`
				ResourceID   string `json:"resource_id"`
				ResourceType string `json:"resource_type"`
			} `json:"findings"`
		}
		json.Unmarshal(evalData, &evalResp)

		// Aggregate by policy
		type policyStats struct {
			Name      string
			Count     int
			Severity  string
			Resources []string
		}

		statsMap := map[string]*policyStats{}
		for _, f := range evalResp.Findings {
			s, exists := statsMap[f.PolicyName]
			if !exists {
				s = &policyStats{Name: f.PolicyName, Severity: f.Severity}
				statsMap[f.PolicyName] = s
			}
			s.Count++
			if len(s.Resources) < 3 {
				s.Resources = append(s.Resources, f.ResourceID)
			}
		}

		// Sort by count desc
		sorted := make([]*policyStats, 0, len(statsMap))
		for _, s := range statsMap {
			sorted = append(sorted, s)
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Count > sorted[j].Count
		})

		// Render
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
		fmt.Println()
		fmt.Println(title.Render("Top Findings by Frequency"))
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
			fmt.Sprintf("  Based on latest evaluation (%d findings)", len(evalResp.Findings))))
		fmt.Println()

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"#", "Policy", "Count", "Severity", "Resources (sample)"})
		table.SetBorder(true)
		table.SetColumnSeparator("│")

		for i, s := range sorted {
			resources := strings.Join(s.Resources, ", ")
			if s.Count > 3 {
				resources += fmt.Sprintf(" +%d", s.Count-3)
			}
			table.Append([]string{
				fmt.Sprintf("%d", i+1),
				s.Name,
				fmt.Sprintf("%d", s.Count),
				s.Severity,
				resources,
			})
		}
		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(topCmd)
}
