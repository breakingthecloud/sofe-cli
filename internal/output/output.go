package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/breakingthecloud/sofe-cli/internal/client"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

var severityIcons = map[string]string{
	"critical": "🔴",
	"high":     "🟠",
	"medium":   "🟡",
	"low":      "🔵",
	"info":     "⚪",
}

func PrintTable(resp *client.EvaluateResponse) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Severity", "Policy", "Resource", "Message"})
	table.SetBorder(false)
	table.SetColumnSeparator("│")

	for _, f := range resp.Findings {
		icon := severityIcons[f.Severity]
		table.Append([]string{
			icon + " " + f.Severity,
			f.PolicyName,
			f.ResourceID,
			f.Message,
		})
	}

	table.Render()
	fmt.Printf("\nSummary: %d findings | Potential savings: $%.2f/mo\n", resp.FindingsCount, resp.TotalEstimatedSavings)
}

func PrintJSON(resp *client.EvaluateResponse) {
	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(data))
}

func PrintMarkdown(resp *client.EvaluateResponse) {
	fmt.Println("# SOFE Evaluation Results")
	fmt.Printf("\n**Findings:** %d | **Savings:** $%.2f/mo\n\n", resp.FindingsCount, resp.TotalEstimatedSavings)
	fmt.Println("| Severity | Policy | Resource | Message |")
	fmt.Println("|----------|--------|----------|---------|")
	for _, f := range resp.Findings {
		fmt.Printf("| %s | %s | %s | %s |\n", f.Severity, f.PolicyName, f.ResourceID, f.Message)
	}
}

func PrintSuccess(msg string) {
	color.Green("✅ %s", msg)
}

func PrintError(msg string) {
	color.Red("❌ %s", msg)
}
