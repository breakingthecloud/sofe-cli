package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	Run: func(cmd *cobra.Command, args []string) {
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Width(12)
		value := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))

		fmt.Println(title.Render(fmt.Sprintf("sofe v%s", Version)))
		fmt.Println()
		fmt.Println(label.Render("Build:") + value.Render(BuildDate))
		fmt.Println(label.Render("Commit:") + value.Render(Commit))
		fmt.Println(label.Render("Go:") + value.Render(runtime.Version()))
		fmt.Println(label.Render("OS/Arch:") + value.Render(runtime.GOOS+"/"+runtime.GOARCH))

		// Config info
		home, _ := os.UserHomeDir()
		configPath := filepath.Join(home, ".sofe", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Println(label.Render("Config:") + value.Render(configPath))
		} else {
			fmt.Println(label.Render("Config:") + value.Render("(not created)"))
		}

		if cfg != nil {
			fmt.Println(label.Render("Mode:") + value.Render(cfg.Mode))
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
