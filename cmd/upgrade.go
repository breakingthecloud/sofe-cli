package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/breakingthecloud/sofe-cli/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [pro]",
	Short: "Upgrade from self-hosted to SOFE Cloud (or to Pro tier)",
	Long:  "Opens browser to sign up for SOFE Cloud platform. After signup, paste your API key to connect the CLI.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		green := color.New(color.FgGreen, color.Bold)
		cyan := color.New(color.FgCyan)

		if len(args) > 0 && args[0] == "pro" {
			// Upgrade to Pro
			green.Println("💰 Upgrade to SOFE Pro")
			fmt.Println()
			fmt.Println("Opening pricing page in your browser...")
			openBrowser("https://platform.sofe.dev/pricing?source=cli")
			fmt.Println()
			fmt.Println("Complete payment there. Your tier upgrades automatically via webhook.")
			fmt.Println("Run 'sofe status' to verify your new tier.")
			return
		}

		// Upgrade from self-hosted to Cloud
		green.Println("🚀 Upgrade to SOFE Cloud (free tier available)")
		fmt.Println()
		fmt.Println("Opening signup page in your browser...")
		openBrowser("https://platform.sofe.dev/signup?source=cli")
		fmt.Println()
		fmt.Println("After signing up:")
		fmt.Println("  1. Go to platform.sofe.dev/keys")
		fmt.Println("  2. Create an API key")
		fmt.Println("  3. Paste it below")
		fmt.Println()

		cyan.Print("API Key: ")
		reader := bufio.NewReader(os.Stdin)
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)

		if key == "" || !strings.HasPrefix(key, "sk_sofe_") {
			color.Red("❌ Invalid key. Must start with 'sk_sofe_'")
			return
		}

		// Save to config
		cfg.APIKey = key
		cfg.Mode = "cloud"
		cfg.CloudURL = "https://api.sofe.dev"
		config.Save(cfg)

		fmt.Println()
		green.Println("✅ Connected! You're now on SOFE Cloud.")
		fmt.Println("  • Use 'sofe evaluate --cloud' for cloud evaluations")
		fmt.Println("  • Your local mode still works unchanged")
		fmt.Println("  • Run 'sofe status' to see your tier")
	},
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
