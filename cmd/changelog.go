package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type githubRelease struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
}

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Show what's new in recent releases",
	Run: func(cmd *cobra.Command, args []string) {
		allFlag, _ := cmd.Flags().GetBool("all")

		limit := 1
		if allFlag {
			limit = 5
		}

		releases, err := fetchReleases(limit)
		if err != nil {
			fmt.Printf("Could not fetch releases: %s\n", err)
			fmt.Println("View online: https://github.com/breakingthecloud/sofe-cli/releases")
			return
		}

		if len(releases) == 0 {
			fmt.Println("No releases found.")
			return
		}

		tagStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
		dateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

		for i, r := range releases {
			date := r.PublishedAt
			if len(date) >= 10 {
				date = date[:10]
			}

			fmt.Printf("%s %s\n", tagStyle.Render(r.TagName), dateStyle.Render("("+date+")"))

			// Format body — indent each line
			body := strings.TrimSpace(r.Body)
			if body == "" {
				body = "  No release notes."
			} else {
				lines := strings.Split(body, "\n")
				var formatted []string
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					// Convert markdown list items
					if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
						formatted = append(formatted, "  "+line[2:])
					} else if strings.HasPrefix(line, "## ") {
						// Skip H2 headers (goreleaser adds "Changelog")
						continue
					} else {
						formatted = append(formatted, "  "+line)
					}
				}
				body = strings.Join(formatted, "\n")
			}

			fmt.Println(bodyStyle.Render(body))

			if i < len(releases)-1 {
				fmt.Println()
			}
		}

		if !allFlag && len(releases) > 0 {
			fmt.Println()
			fmt.Println(dateStyle.Render("  Run 'sofe changelog --all' for more releases"))
		}
	},
}

func fetchReleases(limit int) ([]githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/breakingthecloud/sofe-cli/releases?per_page=%d", limit)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	data, _ := io.ReadAll(resp.Body)
	var releases []githubRelease
	json.Unmarshal(data, &releases)
	return releases, nil
}

func init() {
	changelogCmd.Flags().Bool("all", false, "Show last 5 releases")
	rootCmd.AddCommand(changelogCmd)
}
