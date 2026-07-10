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
			Foreground(lipgloss.Color("10"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14")).
			Background(lipgloss.Color("236"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	sevHighStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("9"))

	sevMedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	sevLowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	panelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238"))

	panelBorderActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("34"))

	panelTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10"))

	detailLabel = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	detailValue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	helpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("234")).
			Padding(0, 1)

	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))
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
	findings      []interactiveFinding
	filtered      []int // indices into findings (after filter)
	cursor        int
	evalID        string
	evalDate      string
	evalTrigger   string
	focus         string // "list" or "detail"
	filter        string
	filterMode    bool
	explanation   string
	explainErr    string
	explaining    bool
	width         int
	height        int
}

type explainMsg struct {
	text string
	err  error
}

func initialSplitModel(evalID, evalDate, trigger string, findings []interactiveFinding) tuiModel {
	indices := make([]int, len(findings))
	for i := range findings {
		indices[i] = i
	}
	return tuiModel{
		findings:    findings,
		filtered:    indices,
		cursor:      0,
		evalID:      evalID,
		evalDate:    evalDate,
		evalTrigger: trigger,
		focus:       "list",
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
		// Filter mode input
		if m.filterMode {
			switch msg.String() {
			case "enter":
				m.filterMode = false
				m.applyFilter()
			case "esc":
				m.filterMode = false
				m.filter = ""
				m.applyFilter()
			case "backspace":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.applyFilter()
				}
			default:
				if len(msg.String()) == 1 {
					m.filter += msg.String()
					m.applyFilter()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.explanation = ""
				m.explainErr = ""
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.explanation = ""
				m.explainErr = ""
			}
		case "g":
			m.cursor = 0
			m.explanation = ""
		case "G":
			m.cursor = len(m.filtered) - 1
			m.explanation = ""
		case "tab":
			if m.focus == "list" {
				m.focus = "detail"
			} else {
				m.focus = "list"
			}
		case "/":
			m.filterMode = true
			m.filter = ""
		case "e":
			if !m.explaining && len(m.filtered) > 0 {
				m.explaining = true
				return m, m.callExplain()
			}
		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.applyFilter()
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

func (m *tuiModel) applyFilter() {
	if m.filter == "" {
		m.filtered = make([]int, len(m.findings))
		for i := range m.findings {
			m.filtered[i] = i
		}
	} else {
		m.filtered = nil
		q := strings.ToLower(m.filter)
		for i, f := range m.findings {
			if strings.Contains(strings.ToLower(f.PolicyName), q) ||
				strings.Contains(strings.ToLower(f.ResourceID), q) ||
				strings.Contains(strings.ToLower(f.Severity), q) ||
				strings.Contains(strings.ToLower(f.ResourceType), q) {
				m.filtered = append(m.filtered, i)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m tuiModel) callExplain() tea.Cmd {
	return func() tea.Msg {
		if len(m.filtered) == 0 {
			return explainMsg{err: fmt.Errorf("no finding selected")}
		}
		aiClient := getAIClient()
		finding := m.findings[m.filtered[m.cursor]]

		reqBody := map[string]interface{}{
			"evaluation_id": m.evalID,
			"finding_index": m.filtered[m.cursor],
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
	if m.width == 0 {
		return "Loading..."
	}
	return m.splitView()
}

func (m tuiModel) splitView() string {
	// Layout: header + [left panel | right panel] + help bar
	totalW := m.width
	leftW := totalW * 40 / 100
	rightW := totalW - leftW - 4 // borders + padding

	if leftW < 30 {
		leftW = 30
	}
	if rightW < 30 {
		rightW = 30
	}

	// Header bar
	headerText := fmt.Sprintf(" SOFE Interactive │ %s │ %d findings │ %s",
		m.evalDate, len(m.findings), m.evalTrigger)
	if m.filter != "" {
		headerText += fmt.Sprintf(" │ filter: %s (%d matches)", m.filter, len(m.filtered))
	}
	header := headerStyle.Width(totalW).Render(headerText)

	// Left panel: findings list
	leftContent := m.renderFindingsList(leftW-4, m.height-6)
	leftBorder := panelBorder
	if m.focus == "list" {
		leftBorder = panelBorderActive
	}
	leftPanel := leftBorder.Width(leftW).Height(m.height - 5).Render(leftContent)

	// Right panel: detail + AI
	rightContent := m.renderDetailPanel(rightW-4, m.height-6)
	rightBorder := panelBorder
	if m.focus == "detail" {
		rightBorder = panelBorderActive
	}
	rightPanel := rightBorder.Width(rightW).Height(m.height - 5).Render(rightContent)

	// Compose
	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// Help bar
	help := helpBarStyle.Render(" ↑↓/jk navigate • e explain • / filter • tab panel • gg/G jump • q quit")
	if m.filterMode {
		help = filterStyle.Render(fmt.Sprintf(" Filter: %s█ (enter confirm, esc cancel)", m.filter))
	}

	return header + "\n" + panels + "\n" + help
}

func (m tuiModel) renderFindingsList(w, h int) string {
	var b strings.Builder

	b.WriteString(panelTitle.Render("FINDINGS"))
	b.WriteString(detailLabel.Render(fmt.Sprintf(" %d/%d", m.cursor+1, len(m.filtered))))
	b.WriteString("\n\n")

	maxShow := h - 3
	if maxShow < 5 {
		maxShow = 10
	}

	start := 0
	if m.cursor >= maxShow {
		start = m.cursor - maxShow + 1
	}

	for i := start; i < len(m.filtered) && i < start+maxShow; i++ {
		f := m.findings[m.filtered[i]]
		sev := sevBadge(f.Severity)
		resource := truncStr(f.ResourceID, 20)
		policy := truncStr(f.PolicyName, w-30)

		line := fmt.Sprintf("%s %s %s", sev, resource, detailLabel.Render(policy))

		if i == m.cursor {
			b.WriteString(selectedStyle.Render("▸" + line))
		} else {
			b.WriteString(normalStyle.Render(" " + line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m tuiModel) renderDetailPanel(w, h int) string {
	if len(m.filtered) == 0 {
		return detailLabel.Render("No findings match filter")
	}

	f := m.findings[m.filtered[m.cursor]]
	var b strings.Builder

	b.WriteString(panelTitle.Render("DETAIL"))
	b.WriteString("\n\n")

	b.WriteString(detailLabel.Render("Policy:    ") + detailValue.Render(f.PolicyName) + "\n")
	b.WriteString(detailLabel.Render("Severity:  ") + sevColor(f.Severity) + "\n")
	b.WriteString(detailLabel.Render("Resource:  ") + detailValue.Render(f.ResourceID) + "\n")
	b.WriteString(detailLabel.Render("Type:      ") + detailValue.Render(f.ResourceType) + "\n")
	b.WriteString(detailLabel.Render("Region:    ") + detailValue.Render(f.Region) + "\n")
	b.WriteString("\n")
	b.WriteString(detailLabel.Render("Message:") + "\n")
	b.WriteString(detailValue.Render("  " + f.Message) + "\n")

	// Remediation
	if len(f.RemediationCommands) > 0 {
		hasCmd := false
		for _, c := range f.RemediationCommands {
			if c != "" {
				hasCmd = true
				break
			}
		}
		if hasCmd {
			b.WriteString("\n")
			b.WriteString(detailLabel.Render("Remediation:") + "\n")
			for _, cmd := range f.RemediationCommands {
				if cmd != "" {
					b.WriteString(detailValue.Render("  $ " + cmd) + "\n")
				}
			}
		}
	}

	// AI section
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("238")).Render("────────────────────────────") + "\n")

	if m.explaining {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("⏳ Calling AI...") + "\n")
	} else if m.explanation != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("AI Explanation") + "\n\n")
		// Truncate explanation to fit panel
		lines := strings.Split(m.explanation, "\n")
		maxLines := h - 16
		if maxLines < 4 {
			maxLines = 4
		}
		for i, line := range lines {
			if i >= maxLines {
				b.WriteString(detailLabel.Render("  ... (scroll with tab+↓)") + "\n")
				break
			}
			b.WriteString(detailValue.Render(wrapLine(line, w-2)) + "\n")
		}
	} else if m.explainErr != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + m.explainErr) + "\n")
	} else {
		b.WriteString(detailLabel.Render("Press 'e' for AI explanation") + "\n")
	}

	return b.String()
}

func sevBadge(sev string) string {
	switch strings.ToLower(sev) {
	case "high", "critical":
		return sevHighStyle.Render("●")
	case "medium":
		return sevMedStyle.Render("○")
	default:
		return sevLowStyle.Render("○")
	}
}

func sevColor(sev string) string {
	switch strings.ToLower(sev) {
	case "high", "critical":
		return sevHighStyle.Render(sev)
	case "medium":
		return sevMedStyle.Render(sev)
	default:
		return sevLowStyle.Render(sev)
	}
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s + strings.Repeat(" ", max-len(s))
	}
	return s[:max-2] + ".."
}

func wrapLine(s string, w int) string {
	if len(s) <= w {
		return s
	}
	return s[:w]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive [eval-id]",
	Short: "Interactive TUI to browse findings",
	Long:  "Launches a split-panel terminal UI: findings list + detail + AI explanation.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Non-TTY detection
		if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
			color.Yellow("sofe interactive requires a terminal (TTY).")
			fmt.Println("  Use 'sofe history' to list evaluations or")
			fmt.Println("  'sofe explain <eval-id> -f <idx>' for AI explanations.")
			return
		}

		// API key check
		if cfg.APIKey == "" {
			color.Yellow("No API key configured.")
			fmt.Println()
			fmt.Println("  Set your key:  sofe config set api-key sk_sofe_xxx")
			fmt.Println("  Or run:        sofe upgrade")
			fmt.Println("  Get a free key: platform.sofe.dev/keys")
			return
		}

		cloudClient := getCloudClient()

		var evalID string
		if len(args) > 0 {
			evalID = args[0]
		} else {
			data, err := cloudClient.Get("/evaluations?limit=1")
			if err != nil {
				color.Red("Error: %s", err)
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

		// Fetch evaluation
		data, err := cloudClient.Get("/evaluations/" + evalID)
		if err != nil {
			color.Red("Error: %s", err)
			return
		}

		var evalResp struct {
			Findings  []interactiveFinding `json:"findings"`
			Timestamp string               `json:"timestamp"`
			Trigger   string               `json:"trigger"`
		}
		json.Unmarshal(data, &evalResp)

		if len(evalResp.Findings) == 0 {
			color.Green("No findings — your infrastructure looks great!")
			return
		}

		trigger := evalResp.Trigger
		if trigger == "" {
			trigger = "manual"
		}
		evalDate := evalResp.Timestamp
		if len(evalDate) > 16 {
			evalDate = evalDate[:16]
		}

		// Launch split-panel TUI
		p := tea.NewProgram(
			initialSplitModel(evalID, evalDate, trigger, evalResp.Findings),
			tea.WithAltScreen(),
		)
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}
