package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/breakingthecloud/sofe-cli/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with SOFE Cloud via browser",
	Long:  "Opens your browser to sign in, then automatically saves your API key.",
	Run: func(cmd *cobra.Command, args []string) {
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		code := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Background(lipgloss.Color("236")).Padding(0, 1)
		success := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))

		fmt.Println()
		fmt.Println(title.Render("SOFE Login"))
		fmt.Println()

		// Step 1: Request device code
		fmt.Println(label.Render("  Requesting device code..."))

		apiURL := cfg.CloudURL
		if apiURL == "" {
			apiURL = "https://api.sofe.dev"
		}

		client := &http.Client{Timeout: 10 * time.Second}
		reqBody := fmt.Sprintf(`{"version":"%s"}`, Version)
		resp, err := client.Post(apiURL+"/auth/device-code", "application/json", 
			strings.NewReader(reqBody))
		if err != nil {
			color.Red("  Failed to connect: %s", err)
			return
		}
		defer resp.Body.Close()

		var deviceResp struct {
			DeviceCode      string `json:"device_code"`
			UserCode        string `json:"user_code"`
			VerificationURL string `json:"verification_url"`
			ExpiresIn       int    `json:"expires_in"`
			Interval        int    `json:"interval"`
		}
		json.NewDecoder(resp.Body).Decode(&deviceResp)

		// Step 2: Open browser
		fmt.Println()
		fmt.Println(label.Render("  Opening browser..."))
		fmt.Println()
		fmt.Println(label.Render("  If it doesn't open, go to:"))
		fmt.Println("  " + lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render(deviceResp.VerificationURL))
		fmt.Println()
		fmt.Println(label.Render("  Enter this code:"))
		fmt.Println("  " + code.Render(deviceResp.UserCode))
		fmt.Println()

		openBrowser(deviceResp.VerificationURL)

		// Step 3: Poll for approval
		spinner := []string{"◐", "◑", "◒", "◓"}
		timeout := time.After(time.Duration(deviceResp.ExpiresIn) * time.Second)
		interval := time.Duration(deviceResp.Interval) * time.Second
		tick := 0

		for {
			select {
			case <-timeout:
				fmt.Println()
				color.Red("  Code expired. Run 'sofe login' again.")
				return
			default:
			}

			fmt.Printf("\r  %s Waiting for confirmation...  ", spinner[tick%4])
			tick++
			time.Sleep(interval)

			// Poll
			pollResp, err := client.Get(apiURL + "/auth/device-poll/" + deviceResp.DeviceCode)
			if err != nil {
				continue
			}

			var pollResult struct {
				Status    string `json:"status"`
				APIKey    string `json:"api_key"`
				UserEmail string `json:"user_email"`
			}
			json.NewDecoder(pollResp.Body).Decode(&pollResult)
			pollResp.Body.Close()

			switch pollResult.Status {
			case "approved":
				fmt.Print("\r")
				fmt.Println(success.Render("  ✓ Logged in as " + pollResult.UserEmail))
				fmt.Println()

				// Save to config
				cfg.APIKey = pollResult.APIKey
				cfg.Mode = "cloud"
				cfg.CloudURL = apiURL
				config.Save(cfg)

				fmt.Println(success.Render("  ✓ API key saved to ~/.sofe/config.yaml"))
				fmt.Println(success.Render("  ✓ Mode set to: cloud"))
				fmt.Println()
				fmt.Println(label.Render("  Next: sofe evaluate --cloud"))
				return

			case "expired":
				fmt.Println()
				color.Red("  Code expired. Run 'sofe login' again.")
				return
			}
			// "pending" — continue polling
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
