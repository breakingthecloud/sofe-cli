package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/breakingthecloud/sofe-cli/internal/client"
	"github.com/breakingthecloud/sofe-cli/internal/output"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	policiesDir   string
	profile       string
	failOn        string
	resourceTypes []string
	autoServe     bool
	cloudMode     bool
	apiKeyFlag    string
)

var (
	stepDone = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	stepRun  = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("◐")
	stepWait = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("○")

	summaryBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")).
			Padding(1, 2)

	summaryTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10"))

	summaryLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Width(14)

	summaryValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	summaryLink = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))
)

func printStep(icon, text string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", icon, text)
}

func clearLine() {
	fmt.Fprint(os.Stderr, "\033[1A\033[2K")
}

func renderSummaryCard(resp *client.EvaluateResponse) string {
	var b strings.Builder

	b.WriteString(summaryTitle.Render("Evaluation Complete"))
	b.WriteString("\n\n")

	// Counts
	high, med, low := 0, 0, 0
	for _, f := range resp.Findings {
		switch strings.ToLower(f.Severity) {
		case "high", "critical":
			high++
		case "medium":
			med++
		default:
			low++
		}
	}

	b.WriteString(summaryLabel.Render("Findings:") + summaryValue.Render(fmt.Sprintf("%d", resp.FindingsCount)) + "\n")
	b.WriteString(summaryLabel.Render("Resources:") + summaryValue.Render(fmt.Sprintf("%d", resp.ResourcesScanned)) + "\n")
	b.WriteString(summaryLabel.Render("Policies:") + summaryValue.Render(fmt.Sprintf("%d", resp.PoliciesEvaluated)) + "\n")
	b.WriteString(summaryLabel.Render("Severity:") + summaryValue.Render(
		fmt.Sprintf("%d high • %d medium • %d low", high, med, low)) + "\n")

	if resp.TotalEstimatedSavings > 0 {
		b.WriteString(summaryLabel.Render("Savings:") + summaryValue.Render(
			fmt.Sprintf("$%.0f/mo", resp.TotalEstimatedSavings)) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(summaryLabel.Render("View:") + summaryLink.Render("platform.sofe.dev/history") + "\n")
	b.WriteString(summaryLabel.Render("Next:") + summaryLink.Render("sofe interactive") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(" • ") +
		summaryLink.Render(fmt.Sprintf("sofe explain %s -f 0", resp.EvaluationID[:8])) + "\n")
	b.WriteString(summaryLabel.Render("ID:") + summaryValue.Render(resp.EvaluationID) + "\n")

	return summaryBox.Render(b.String())
}

var evaluateCmd = &cobra.Command{
	Use:   "evaluate",
	Short: "Evaluate policies against live AWS resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := policiesDir
		if dir == "" {
			dir = cfg.PoliciesDir
		}
		prof := profile
		if prof == "" {
			prof = cfg.AWSProfile
		}

		// Determine mode: --cloud flag > config mode
		useCloud := cloudMode || cfg.Mode == "cloud"

		if useCloud {
			// Cloud mode resolves the AWS account server-side (connected account / Lambda role).
			// The local AWS profile is meaningless to the API — sending it causes ProfileNotFound.
			return evaluateCloud(dir, "")
		}
		return evaluateLocal(dir, prof)
	},
}

func evaluateCloud(dir, prof string) error {
	key := apiKeyFlag
	if key == "" {
		key = cfg.APIKey
	}
	if key == "" {
		output.PrintError("Cloud mode requires API key. Use --api-key or run: sofe config set api-key <your-key>")
		os.Exit(1)
	}

	cloudClient := client.New(cfg.CloudURL, key)

	// Animated progress steps
	fmt.Println()
	printStep(stepRun, "Connecting to api.sofe.dev...")

	start := time.Now()

	// Actually call the API (single call, but we show steps for UX)
	resp, err := cloudClient.Evaluate(client.EvaluateRequest{
		PoliciesDir:   dir,
		Profile:       prof,
		ResourceTypes: resourceTypes,
		FailOn:        failOn,
	})

	elapsed := time.Since(start)

	if err != nil {
		clearLine()
		printStep(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗"), "Connection failed")
		fmt.Println()
		output.PrintError(err.Error())
		return err
	}

	// Replace progress with completed steps
	clearLine()
	printStep(stepDone, "Connected to api.sofe.dev")
	printStep(stepDone, fmt.Sprintf("Scanned %d resources (17 collectors)", resp.ResourcesScanned))
	printStep(stepDone, fmt.Sprintf("Evaluated %d policies", resp.PoliciesEvaluated))
	printStep(stepDone, fmt.Sprintf("Complete — %d findings in %.1fs", resp.FindingsCount, elapsed.Seconds()))
	fmt.Println()

	// Summary card
	fmt.Println(renderSummaryCard(resp))
	fmt.Println()

	// Show table/json/markdown based on format
	format := formatFlag
	if format == "" {
		format = cfg.DefaultFormat
	}
	switch format {
	case "json":
		output.PrintJSON(resp)
	case "markdown":
		output.PrintMarkdown(resp)
	default:
		output.PrintTable(resp)
	}

	if resp.Failed {
		fmt.Printf("\n❌ FAILED: findings at or above '%s' severity\n", failOn)
		os.Exit(1)
	}
	return nil
}

func evaluateLocal(dir, prof string) error {
	shouldStop := false
	if autoServe {
		shouldStop = AutoServe(cfg.APIURL)
	}
	defer func() {
		if shouldStop {
			fmt.Println("\n⏹ Stopping auto-started server...")
			stopServer()
		}
	}()

	fmt.Println()
	printStep(stepRun, fmt.Sprintf("Evaluating policies from: %s", dir))

	start := time.Now()
	resp, err := apiClient.Evaluate(client.EvaluateRequest{
		PoliciesDir:   dir,
		Profile:       prof,
		ResourceTypes: resourceTypes,
		FailOn:        failOn,
	})
	elapsed := time.Since(start)

	if err != nil {
		clearLine()
		printStep(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗"), "Evaluation failed")
		fmt.Println()
		output.PrintError(err.Error())
		return err
	}

	clearLine()
	printStep(stepDone, fmt.Sprintf("Scanned %d resources", resp.ResourcesScanned))
	printStep(stepDone, fmt.Sprintf("Evaluated — %d findings in %.1fs", resp.FindingsCount, elapsed.Seconds()))
	fmt.Println()

	format := formatFlag
	if format == "" {
		format = cfg.DefaultFormat
	}
	switch format {
	case "json":
		output.PrintJSON(resp)
	case "markdown":
		output.PrintMarkdown(resp)
	default:
		output.PrintTable(resp)
	}

	if resp.Failed {
		fmt.Printf("\n❌ FAILED: findings at or above '%s' severity\n", failOn)
		os.Exit(1)
	}
	return nil
}

func init() {
	evaluateCmd.Flags().StringVarP(&policiesDir, "policies", "p", "", "Policies directory")
	evaluateCmd.Flags().StringVar(&profile, "profile", "", "AWS profile")
	evaluateCmd.Flags().StringVar(&failOn, "fail-on", "", "Fail if findings at/above severity (critical|high|medium|low)")
	evaluateCmd.Flags().StringSliceVar(&resourceTypes, "resource-types", nil, "Filter resource types")
	evaluateCmd.Flags().BoolVar(&autoServe, "auto-serve", false, "Auto-start server if not running (stops after evaluation)")
	evaluateCmd.Flags().BoolVar(&cloudMode, "cloud", false, "Use cloud API (api.sofe.dev) instead of local server")
	evaluateCmd.Flags().StringVar(&apiKeyFlag, "api-key", "", "API key for cloud mode (sk_sofe_xxx)")
	rootCmd.AddCommand(evaluateCmd)
}
