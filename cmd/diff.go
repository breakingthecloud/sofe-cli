package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <eval-id-1> <eval-id-2>",
	Short: "Compare two evaluations (new/fixed/unchanged findings)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if cfg.APIKey == "" {
			color.Yellow("No API key configured.")
			return
		}

		id1, id2 := args[0], args[1]
		cloudClient := getCloudClient()

		// Fetch both evaluations
		data1, err := cloudClient.Get("/evaluations/" + id1)
		if err != nil {
			color.Red("Error fetching eval 1: %s", err)
			return
		}
		data2, err := cloudClient.Get("/evaluations/" + id2)
		if err != nil {
			color.Red("Error fetching eval 2: %s", err)
			return
		}

		type finding struct {
			PolicyName string `json:"policy_name"`
			ResourceID string `json:"resource_id"`
			Severity   string `json:"severity"`
			Message    string `json:"message"`
		}
		type evalData struct {
			Findings  []finding `json:"findings"`
			Timestamp string    `json:"timestamp"`
		}

		var eval1, eval2 evalData
		json.Unmarshal(data1, &eval1)
		json.Unmarshal(data2, &eval2)

		// Build key sets (policy+resource = unique finding)
		keyOf := func(f finding) string {
			return f.PolicyName + "|" + f.ResourceID
		}

		set1 := map[string]finding{}
		for _, f := range eval1.Findings {
			set1[keyOf(f)] = f
		}
		set2 := map[string]finding{}
		for _, f := range eval2.Findings {
			set2[keyOf(f)] = f
		}

		// Compute diff
		var newFindings, fixedFindings []finding
		unchanged := 0

		for k, f := range set2 {
			if _, exists := set1[k]; !exists {
				newFindings = append(newFindings, f)
			} else {
				unchanged++
			}
		}
		for k, f := range set1 {
			if _, exists := set2[k]; !exists {
				fixedFindings = append(fixedFindings, f)
			}
		}

		// Render
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		added := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		fixed := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		neutral := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

		date1 := eval1.Timestamp
		if len(date1) > 16 {
			date1 = date1[:16]
		}
		date2 := eval2.Timestamp
		if len(date2) > 16 {
			date2 = date2[:16]
		}

		fmt.Println()
		fmt.Println(title.Render("Evaluation Diff"))
		fmt.Println(label.Render(fmt.Sprintf("  A: %s (%d findings) — %s", id1[:8], len(eval1.Findings), date1)))
		fmt.Println(label.Render(fmt.Sprintf("  B: %s (%d findings) — %s", id2[:8], len(eval2.Findings), date2)))
		fmt.Println()

		// Summary
		fmt.Printf("  %s  %s  %s\n",
			added.Render(fmt.Sprintf("+%d new", len(newFindings))),
			fixed.Render(fmt.Sprintf("-%d fixed", len(fixedFindings))),
			neutral.Render(fmt.Sprintf("=%d unchanged", unchanged)),
		)
		fmt.Println()

		// New findings (in B but not A = new problems)
		if len(newFindings) > 0 {
			fmt.Println(added.Render("  + New findings (appeared in B):"))
			for _, f := range newFindings {
				sev := strings.ToUpper(f.Severity[:1])
				fmt.Printf("    %s %s  %s\n",
					added.Render(sev),
					lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(f.ResourceID),
					label.Render(f.PolicyName),
				)
			}
			fmt.Println()
		}

		// Fixed findings (in A but not B = resolved)
		if len(fixedFindings) > 0 {
			fmt.Println(fixed.Render("  - Fixed findings (resolved in B):"))
			for _, f := range fixedFindings {
				sev := strings.ToUpper(f.Severity[:1])
				fmt.Printf("    %s %s  %s\n",
					fixed.Render(sev),
					lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(f.ResourceID),
					label.Render(f.PolicyName),
				)
			}
			fmt.Println()
		}

		if len(newFindings) == 0 && len(fixedFindings) == 0 {
			fmt.Println(neutral.Render("  No changes between evaluations."))
		}
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
