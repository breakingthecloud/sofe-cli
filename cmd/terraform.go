package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/breakingthecloud/sofe-cli/internal/terraform"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var terraformCmd = &cobra.Command{
	Use:   "terraform <path>",
	Short: "Scan Terraform plans/files for FinOps policy violations (pre-deploy)",
	Long: `Pre-deploy scan of Terraform infrastructure:

  sofe terraform ./tfplan.json     # scan a plan JSON (terraform show -json)
  sofe terraform ./infra/          # scan a directory of .tf files

Evaluates FinOps policies BEFORE you apply — catch issues in PRs, not production.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		failOn, _ := cmd.Flags().GetString("fail-on")

		start := time.Now()

		// Detect mode: file (plan JSON) or directory (.tf files)
		info, err := os.Stat(path)
		if err != nil {
			color.Red("  Error: %s", err)
			return
		}

		fmt.Println()

		var resources []terraform.Resource

		if info.IsDir() {
			printStep(stepRun, fmt.Sprintf("Scanning .tf files in %s...", path))
			resources, err = terraform.ParseDirectory(path)
			if err == nil {
				clearLine()
				printStep(stepDone, fmt.Sprintf("Parsed %d planned resources (mode: directory — basic detection)", len(resources)))
				fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(
					"  ⚠ For accurate results, use: terraform show -json tfplan > plan.json"))
			}
		} else if strings.HasSuffix(path, ".json") {
			printStep(stepRun, fmt.Sprintf("Parsing plan JSON: %s...", filepath.Base(path)))
			resources, err = terraform.ParsePlanJSON(path)
			if err == nil {
				clearLine()
				printStep(stepDone, fmt.Sprintf("Parsed %d planned resources (mode: plan — deterministic)", len(resources)))
			}
		} else {
			color.Red("  Error: path must be a directory (with .tf files) or a .json plan file")
			return
		}

		if err != nil {
			clearLine()
			printStep(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗"), err.Error())
			return
		}

		if len(resources) == 0 {
			printStep(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("!"), "No resources found in the plan")
			return
		}

		// Evaluate policies
		policies := terraform.DefaultPolicies()
		printStep(stepRun, fmt.Sprintf("Evaluating %d pre-deploy policies...", len(policies)))
		findings := terraform.Evaluate(resources, policies)
		elapsed := time.Since(start)

		clearLine()
		printStep(stepDone, fmt.Sprintf("Evaluated %d policies", len(policies)))
		printStep(stepDone, fmt.Sprintf("Complete — %d findings in %.1fs", len(findings), elapsed.Seconds()))
		fmt.Println()

		// Summary card
		high, med, low, crit := 0, 0, 0, 0
		for _, f := range findings {
			switch f.Severity {
			case "critical":
				crit++
			case "high":
				high++
			case "medium":
				med++
			default:
				low++
			}
		}

		summaryStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(1, 2)

		if len(findings) == 0 {
			fmt.Println(summaryStyle.BorderForeground(lipgloss.Color("10")).Render(
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")).Render("Pre-Deploy Check Passed") + "\n\n" +
					fmt.Sprintf("  %d resources • %d policies • 0 findings\n", len(resources), len(policies)) +
					"  Safe to apply!"))
		} else {
			titleColor := lipgloss.Color("11")
			if crit > 0 || high > 0 {
				titleColor = lipgloss.Color("9")
			}

			var sb strings.Builder
			sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(titleColor).Render("Pre-Deploy Findings"))
			sb.WriteString("\n\n")
			sb.WriteString(fmt.Sprintf("  %d resources • %d policies • %d findings\n", len(resources), len(policies), len(findings)))
			sb.WriteString(fmt.Sprintf("  Severity: %d critical • %d high • %d medium • %d low\n", crit, high, med, low))

			fmt.Println(summaryStyle.BorderForeground(titleColor).Render(sb.String()))
		}

		fmt.Println()

		// Findings table
		if len(findings) > 0 {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Severity", "Policy", "Resource", "Message"})
			table.SetBorder(true)
			table.SetColumnSeparator("│")
			table.SetColWidth(40)

			for _, f := range findings {
				sev := f.Severity
				switch sev {
				case "critical":
					sev = "🔴 CRIT"
				case "high":
					sev = "🟠 HIGH"
				case "medium":
					sev = "🟡 MED"
				default:
					sev = "🟢 LOW"
				}
				table.Append([]string{sev, f.PolicyName, f.ResourceID, f.Message})
			}
			table.Render()
		}

		// Fail-on check
		if failOn != "" {
			shouldFail := false
			switch strings.ToLower(failOn) {
			case "critical":
				shouldFail = crit > 0
			case "high":
				shouldFail = crit > 0 || high > 0
			case "medium":
				shouldFail = crit > 0 || high > 0 || med > 0
			case "low":
				shouldFail = len(findings) > 0
			}
			if shouldFail {
				fmt.Printf("\n❌ FAILED: findings at or above '%s' severity — DO NOT APPLY\n", failOn)
				os.Exit(1)
			}
		}

		if len(findings) > 0 {
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
				"  Fix these before running 'terraform apply'"))
			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(
				"  Use --fail-on high in CI/CD to block deploys"))
		}
	},
}

func init() {
	terraformCmd.Flags().String("fail-on", "", "Exit 1 if findings at/above severity (critical|high|medium|low)")
	rootCmd.AddCommand(terraformCmd)
}
