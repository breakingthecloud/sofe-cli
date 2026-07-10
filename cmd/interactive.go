package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
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

	menuBoxStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10"))
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
	evalDate     string
	evalTrigger  string
	view         string // "menu", "list", "detail"
	menuCursor   int
	explanation  string
	explainErr   string
	explaining   bool
	width        int
	height       int
}

type explainMsg struct {
	text string
	err  error
}

func initialModel(evalID, evalDate, trigger string, findings []interactiveFinding) tuiModel {
	return tuiModel{
		findings:    findings,
		cursor:      0,
		evalID:      evalID,
		evalDate:    evalDate,
		evalTrigger: trigger,
		view:        "menu",
		menuCursor:  0,
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
			if m.view == "menu" && m.menuCursor > 0 {
				m.menuCursor--
			} else if m.view == "list" && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.view == "menu" && m.menuCursor < 3 {
				m.menuCursor++
			} else if m.view == "list" && m.cursor < len(m.findings)-1 {
				m.cursor++
			}
		case "enter":
			if m.view == "menu" {
				switch m.menuCursor {
				case 0: // Browse findings
					m.view = "list"
				case 1: // Severity breakdown (inline, no action needed)
				case 2: // Account status (inline, no action needed)
				case 3: // Quit
					return m, tea.Quit
				}
			} else if m.view == "list" {
				m.view = "detail"
				m.explanation = ""
				m.explainErr = ""
				m.explaining = false
			}
		case "esc", "backspace":
			if m.view == "detail" {
				m.view = "list"
				m.explanation = ""
				m.explainErr = ""
			} else if m.view == "list" {
				m.view = "menu"
			}
		case "e":
			if m.view == "detail" && !m.explaining {
				m.explaining = true
				return m, m.callExplain()
			}
		}

	case explainMsg:
		m.explaining = false
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
	switch m.view {
	case "menu":
		return m.menuView()
	case "detail":
		return m.detailView()
	default:
		return m.listView()
	}
}

func (m tuiModel) menuView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("SOFE Interactive"))
	b.WriteString("\n\n")

	// Evaluation summary
	summary := fmt.Sprintf(
		"  Evaluation:  %s\n  Trigger:     %s\n  Findings:    %d\n  ID:          %s",
		m.evalDate, m.evalTrigger, len(m.findings), m.evalID[:8]+"...",
	)
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(summary))
	b.WriteString("\n\n")

	// Menu items
	menuItems := []string{
		fmt.Sprintf("Browse findings (%d)", len(m.findings)),
		"View severity breakdown",
		"Account status",
		"Quit",
	}

	for i, item := range menuItems {
		if i == m.menuCursor {
			b.WriteString(selectedStyle.Render("▸ " + item))
		} else {
			b.WriteString(normalStyle.Render("  " + item))
		}
		b.WriteString("\n")
	}

	// Show inline content based on selection
	if m.menuCursor == 1 {
		b.WriteString("\n")
		high, med, low := 0, 0, 0
		for _, f := range m.findings {
			switch strings.ToLower(f.Severity) {
			case "high", "critical":
				high++
			case "medium":
				med++
			default:
				low++
			}
		}
		breakdown := fmt.Sprintf("  HIGH: %d  |  MEDIUM: %d  |  LOW: %d", high, med, low)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(breakdown))

		// Top policies
		policyCount := map[string]int{}
		for _, f := range m.findings {
			policyCount[f.PolicyName]++
		}
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  Top policies:"))
		b.WriteString("\n")
		shown := 0
		for policy, count := range policyCount {
			if shown >= 5 {
				break
			}
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(fmt.Sprintf("    %s (%d)", policy, count)))
			b.WriteString("\n")
			shown++
		}
	} else if m.menuCursor == 2 {
		b.WriteString("\n")
		statusInfo := fmt.Sprintf("  Mode:     cloud\n  API Key:  %s...%s\n  Findings: %d in latest eval",
			cfg.APIKey[:12], cfg.APIKey[len(cfg.APIKey)-4:], len(m.findings))
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(statusInfo))
	}

	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("↑/↓ select • enter confirm • q quit"))

	return b.String()
}

func (m tuiModel) listView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("Findings — %d total", len(m.findings))))
	b.WriteString("\n\n")

	maxShow := m.height - 6
	if maxShow < 5 {
		maxShow = 15
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
	b.WriteString(helpStyle.Render("↑/↓ navigate • enter detail • esc menu • q quit"))

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
			if cmd != "" {
				detail += fmt.Sprintf("\n  %d. %s", i+1, cmd)
			}
		}
	}

	b.WriteString(detailStyle.Render(detail))

	if m.explaining {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("⏳ Calling AI..."))
	} else if m.explanation != "" {
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
	b.WriteString(helpStyle.Render("e explain (AI) • esc back • q quit"))

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
		// Non-TTY detection
		if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			color.Yellow("⚠️  sofe interactive requires a terminal (TTY).")
			fmt.Println("  Use 'sofe history' to list evaluations or")
			fmt.Println("  'sofe explain <eval-id> -f <idx>' for AI explanations.")
			return
		}

		// API key check — don't force upgrade, just inform
		if cfg.APIKey == "" {
			color.Yellow("⚠️  No API key configured.")
			fmt.Println()
			fmt.Println("  Set your key:  sofe config set api-key sk_sofe_xxx")
			fmt.Println("  Or run:        sofe upgrade")
			fmt.Println()
			fmt.Println("  Get a free key at: platform.sofe.dev/keys")
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
			Findings  []interactiveFinding `json:"findings"`
			Timestamp string               `json:"timestamp"`
			Trigger   string               `json:"trigger"`
		}
		json.Unmarshal(data, &evalResp)

		if len(evalResp.Findings) == 0 {
			color.Green("✅ No findings in evaluation %s — your infrastructure looks great!", evalID)
			return
		}

		trigger := evalResp.Trigger
		if trigger == "" {
			trigger = "manual"
		}
		evalDate := evalResp.Timestamp
		if len(evalDate) > 16 {
			evalDate = evalDate[:16] // trim to "2026-07-09T06:00"
		}

		// Launch TUI with menu
		p := tea.NewProgram(initialModel(evalID, evalDate, trigger, evalResp.Findings), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}
