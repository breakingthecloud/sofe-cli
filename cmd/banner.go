package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	bannerBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("34")).
			Padding(1, 2)

	bannerTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15"))

	bannerVersion = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	bannerLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Width(12)

	bannerValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	bannerCmd = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))

	bannerDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	bannerSection = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")).
			MarginTop(1)
)

func renderWelcomeBanner() string {
	var b strings.Builder

	// Title
	b.WriteString(bannerTitle.Render("SOFE — Stairway Open FinOps Engine"))
	b.WriteString("\n")
	b.WriteString(bannerVersion.Render("v" + Version))
	b.WriteString("\n")

	// Status section
	if cfg != nil && cfg.APIKey != "" {
		b.WriteString("\n")
		mode := "cloud"
		if cfg.Mode == "local" {
			mode = "local (" + cfg.APIURL + ")"
		}
		keyDisplay := cfg.APIKey
		if len(keyDisplay) > 16 {
			keyDisplay = keyDisplay[:12] + "..." + keyDisplay[len(keyDisplay)-4:]
		}

		b.WriteString(bannerLabel.Render("Mode:") + bannerValue.Render(mode) + "\n")
		b.WriteString(bannerLabel.Render("API Key:") + bannerValue.Render(keyDisplay) + "\n")

		// Try to get latest eval (non-blocking, best effort)
		lastEval := getLastEvalSummary()
		if lastEval != "" {
			b.WriteString(bannerLabel.Render("Last eval:") + bannerValue.Render(lastEval) + "\n")
		}
	} else {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("No API key configured") + "\n")
	}

	// Commands section
	b.WriteString("\n")
	b.WriteString(bannerSection.Render("Commands"))
	b.WriteString("\n")

	commands := []struct{ cmd, desc string }{
		{"evaluate", "Run policies against AWS resources"},
		{"interactive", "Browse findings in TUI"},
		{"history", "List past evaluations"},
		{"explain", "AI explanation for a finding"},
		{"remediate", "Show/run fix commands"},
		{"status", "Account & tier info"},
		{"changelog", "What's new in this version"},
		{"upgrade", "Connect to cloud or upgrade tier"},
	}

	for _, c := range commands {
		b.WriteString("  " + bannerCmd.Render(fmt.Sprintf("%-14s", c.cmd)) + bannerDesc.Render(c.desc) + "\n")
	}

	// Quick start
	b.WriteString("\n")
	b.WriteString(bannerSection.Render("Quick Start"))
	b.WriteString("\n")

	if cfg != nil && cfg.APIKey != "" {
		b.WriteString(bannerDesc.Render("  sofe evaluate --cloud") + bannerDesc.Render("        Scan now") + "\n")
		b.WriteString(bannerDesc.Render("  sofe interactive") + bannerDesc.Render("             Full TUI") + "\n")
		b.WriteString(bannerDesc.Render("  sofe explain <id> -f 0") + bannerDesc.Render("       Explain finding") + "\n")
	} else {
		b.WriteString(bannerDesc.Render("  sofe upgrade") + bannerDesc.Render("       Sign in & get API key") + "\n")
		b.WriteString(bannerDesc.Render("  sofe serve") + bannerDesc.Render("         Run locally (no account)") + "\n")
	}

	content := b.String()
	return bannerBorder.Render(content)
}

func renderFirstRunBanner() string {
	var b strings.Builder

	b.WriteString(bannerTitle.Render("SOFE — Stairway Open FinOps Engine"))
	b.WriteString("\n")
	b.WriteString(bannerVersion.Render("v" + Version))
	b.WriteString("\n\n")

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render("Welcome! Choose how to get started:"))
	b.WriteString("\n\n")

	b.WriteString(bannerSection.Render("Cloud (recommended)"))
	b.WriteString("\n")
	b.WriteString(bannerDesc.Render("  1. sofe upgrade") + bannerDesc.Render("          Sign in & get API key") + "\n")
	b.WriteString(bannerDesc.Render("  2. sofe evaluate --cloud") + bannerDesc.Render("   Run your first scan") + "\n")
	b.WriteString("\n")

	b.WriteString(bannerSection.Render("Self-hosted"))
	b.WriteString("\n")
	b.WriteString(bannerDesc.Render("  1. sofe serve") + bannerDesc.Render("            Start local server") + "\n")
	b.WriteString(bannerDesc.Render("  2. sofe evaluate") + bannerDesc.Render("         Scan with local credentials") + "\n")
	b.WriteString("\n")

	b.WriteString(bannerDesc.Render("  Docs: sofe.dev/docs") + "\n")
	b.WriteString(bannerDesc.Render("  GitHub: github.com/breakingthecloud/sofe-cli") + "\n")

	content := b.String()
	return bannerBorder.Render(content)
}

func getLastEvalSummary() string {
	if cfg == nil || cfg.APIKey == "" {
		return ""
	}
	client := getCloudClient()
	data, err := client.Get("/evaluations?limit=1")
	if err != nil {
		return ""
	}
	var resp HistoryResponse
	json.Unmarshal(data, &resp)
	if len(resp.Evaluations) == 0 {
		return "none"
	}
	e := resp.Evaluations[0]
	trigger := e.Trigger
	if trigger == "" {
		trigger = "manual"
	}
	date := e.Timestamp
	if len(date) > 10 {
		date = date[:10]
	}
	return fmt.Sprintf("%d findings (%s, %s)", e.FindingsCount, date, trigger)
}
