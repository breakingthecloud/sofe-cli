package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor evaluations in real-time (refresh periodically)",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg.APIKey == "" {
			color.Yellow("No API key configured.")
			return
		}

		once, _ := cmd.Flags().GetBool("once")
		interval, _ := cmd.Flags().GetDuration("interval")

		cloudClient := getCloudClient()

		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		value := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("34")).
			Padding(1, 2)

		renderWatch := func() bool {
			// Get recent evaluations for trend
			data, err := cloudClient.Get("/evaluations?limit=7")
			if err != nil {
				color.Red("Error: %s", err)
				return false
			}
			var hist HistoryResponse
			json.Unmarshal(data, &hist)

			if len(hist.Evaluations) == 0 {
				fmt.Println("No evaluations yet.")
				return false
			}

			// Build sparkline
			counts := make([]int, len(hist.Evaluations))
			for i, e := range hist.Evaluations {
				counts[len(hist.Evaluations)-1-i] = e.FindingsCount // reverse for chronological
			}
			sparkline := renderSparkline(counts)

			// Trend
			trend := "stable"
			if len(counts) >= 2 {
				first := counts[0]
				last := counts[len(counts)-1]
				if last < first-2 {
					trend = "improving ↓"
				} else if last > first+2 {
					trend = "degrading ↑"
				}
			}

			latest := hist.Evaluations[0]
			latestDate := latest.Timestamp
			if len(latestDate) > 16 {
				latestDate = latestDate[:16]
			}

			// Changes from previous
			changes := ""
			if len(hist.Evaluations) >= 2 {
				diff := latest.FindingsCount - hist.Evaluations[1].FindingsCount
				if diff > 0 {
					changes = fmt.Sprintf("+%d new since last", diff)
				} else if diff < 0 {
					changes = fmt.Sprintf("%d fixed since last", -diff)
				} else {
					changes = "no change since last"
				}
			}

			// Clear screen for live mode
			if !once {
				fmt.Print("\033[H\033[2J")
			}

			var b strings.Builder
			b.WriteString(title.Render("SOFE Watch") + label.Render("  live") + "\n\n")
			b.WriteString(label.Render("Findings:  ") + value.Render(fmt.Sprintf("%d", latest.FindingsCount)))
			b.WriteString("  " + value.Render(sparkline))
			b.WriteString("  " + label.Render("("+trend+")") + "\n")
			b.WriteString(label.Render("Last:      ") + value.Render(latestDate) + "\n")
			b.WriteString(label.Render("Resources: ") + value.Render(fmt.Sprintf("%d", latest.ResourcesScanned)) + "\n")
			if changes != "" {
				b.WriteString(label.Render("Changes:   ") + value.Render(changes) + "\n")
			}
			if !once {
				b.WriteString("\n" + label.Render(fmt.Sprintf("Refreshing every %s • Ctrl+C to exit", interval)))
			}

			fmt.Println(box.Render(b.String()))
			return true
		}

		if once {
			renderWatch()
			return
		}

		// Live mode
		fmt.Println(title.Render("Starting watch mode..."))
		for {
			if !renderWatch() {
				return
			}
			time.Sleep(interval)
		}
	},
}

func renderSparkline(values []int) string {
	if len(values) == 0 {
		return ""
	}
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	var spark strings.Builder
	for _, v := range values {
		idx := 0
		if max > min {
			idx = (v - min) * (len(blocks) - 1) / (max - min)
		}
		spark.WriteRune(blocks[idx])
	}
	return spark.String()
}

func init() {
	watchCmd.Flags().Bool("once", false, "Show status once and exit (no live refresh)")
	watchCmd.Flags().Duration("interval", 5*time.Minute, "Refresh interval (e.g. 30s, 5m, 1h)")
	rootCmd.AddCommand(watchCmd)
}
