package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")).
			Background(lipgloss.Color("236"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	severityHighStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("9"))

	severityMediumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11"))

	severityLowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10"))

	detailStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type interactiveFinding struct {
	PolicyName          string   `json:"policy_name"`
	Severity            string   `json:"severity"`
	ResourceID          string   `json:"resource_id"`
	ResourceType        string   `json:"resource_type"`
	Region              string   `json:"region"`
	Message             string   `json:"message"`
	RemediationCommands []string `json:"remediation_commands"`
}

type tuiModel struct {
	findings     []interactiveFinding
	cursor       int
	evalID       string
	view         string // "list" or "detail"
	explanation  string
	explainErr   string
	width        int
	height       int
}

type explainMsg struct {
	text string
	err  error
}

func initialModel(evalID string, findings []interactiveFinding) tuiModel {
	return tuiModel{
		findings: findings,
		cursor:   0,
		evalID:   evalID,
		view:     "list",
	}
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.view == "list" && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.view == "list" && m.cursor < len(m.findings)-1 {
				m.cursor++
			}
		case "enter":
			if m.view == "list" {
				m.view = "detail"
				m.explanation = ""
				m.explainErr = ""
			}
		case "esc", "backspace":
			if m.view == "detail" {
				m.view = "list"
			}
		case "e":
			if m.view == "detail" {
				return m, m.callExplain()
			}
		}

	case explainMsg:
		if msg.err != nil {
			m.explainErr = msg.err.Error()
		} else {
			m.explanation = msg.text
		}
	}

	return m, nil
}

func (m tuiModel) callExplain() tea.Cmd {
	return func() tea.Msg {
		aiClient := getAIClient()
		finding := m.findings[m.cursor]

		reqBody := map[string]interface{}{
			"evaluation_id": m.evalID,
			"finding_index": m.cursor,
			"finding": map[string]string{
				"policy_name":   finding.PolicyName,
				"severity":      finding.Severity,
				"resource_id":   finding.ResourceID,
				"resource_type": finding.ResourceType,
				"message":       finding.Message,
			},
		}

		data, err := aiClient.Post("/explain", reqBody)
		if err != nil {
			return explainMsg{err: err}
		}

		var resp struct {
			Explanation string `json:"explanation"`
		}
		json.Unmarshal(data, &resp)
		return explainMsg{text: resp.Explanation}
	}
}

func (m tuiModel) View() string {
	if m.view == "detail" {
		return m.detailView()
	}
	return m.listView()
}

func (m tuiModel) listView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("🔍 SOFE Interactive — %d findings", len(m.findings))))
	b.WriteString("\n\n")

	maxShow := m.height - 6
	if maxShow < 5 {
		maxShow = 10
	}
	start := 0
	if m.cursor >= maxShow {
		start = m.cursor - maxShow + 1
	}

	for i := start; i < len(m.findings) && i < start+maxShow; i++ {
		f := m.findings[i]
		sevBadge := severityBadge(f.Severity)
		line := fmt.Sprintf(" %s  %-25s %s", sevBadge, truncate(f.ResourceID, 25), f.PolicyName)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render("▸ " + line))
		} else {
			b.WriteString(normalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate • enter detail • q quit"))

	return b.String()
}

func (m tuiModel) detailView() string {
	f := m.findings[m.cursor]
	var b strings.Builder

	header := fmt.Sprintf("Finding #%d — %s", m.cursor, f.PolicyName)
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n\n")

	detail := fmt.Sprintf(
		"Severity:  %s\nResource:  %s\nType:      %s\nRegion:    %s\n\nMessage:\n  %s",
		f.Severity, f.ResourceID, f.ResourceType, f.Region, f.Message,
	)

	if len(f.RemediationCommands) > 0 {
		detail += "\n\nRemediation:"
		for i, cmd := range f.RemediationCommands {
			detail += fmt.Sprintf("\n  %d. %s", i+1, cmd)
		}
	}

	b.WriteString(detailStyle.Render(detail))

	if m.explanation != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("🤖 AI Explanation:"))
		b.WriteString("\n")
		b.WriteString(m.explanation)
	}
	if m.explainErr != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("❌ " + m.explainErr))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("e explain • esc back • q quit"))

	return b.String()
}

func severityBadge(sev string) string {
	switch strings.ToLower(sev) {
	case "high", "critical":
		return severityHighStyle.Render("HIGH")
	case "medium":
		return severityMediumStyle.Render(" MED")
	default:
		return severityLowStyle.Render(" LOW")
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s + strings.Repeat(" ", max-len(s))
	}
	return s[:max-3] + "..."
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive [eval-id]",
	Short: "Interactive TUI to browse findings",
	Long:  "Launches a terminal UI to navigate findings, view details, and get AI explanations.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if cfg.APIKey == "" {
			color.Red("❌ No API key configured. Run 'sofe upgrade' first.")
			return
		}

		cloudClient := getCloudClient()

		var evalID string
		if len(args) > 0 {
			evalID = args[0]
		} else {
			// Get latest evaluation
			data, err := cloudClient.Get("/evaluations?limit=1")
			if err != nil {
				color.Red("❌ %s", err)
				return
			}
			var hist HistoryResponse
			json.Unmarshal(data, &hist)
			if len(hist.Evaluations) == 0 {
				color.Yellow("No evaluations found. Run 'sofe evaluate --cloud' first.")
				return
			}
			evalID = hist.Evaluations[0].ID
		}

		// Fetch evaluation findings
		data, err := cloudClient.Get("/evaluations/" + evalID)
		if err != nil {
			color.Red("❌ %s", err)
			return
		}

		var evalResp struct {
			Findings []interactiveFinding `json:"findings"`
		}
		json.Unmarshal(data, &evalResp)

		if len(evalResp.Findings) == 0 {
			color.Green("✅ No findings in evaluation %s — your infrastructure looks great!", evalID)
			return
		}

		// Launch TUI
		p := tea.NewProgram(initialModel(evalID, evalResp.Findings), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}
