package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var remediateCmd = &cobra.Command{
	Use:   "remediate <eval-id>",
	Short: "Show/execute remediation commands for a finding",
	Long:  "Displays the CLI commands to fix a finding. With --execute, runs them interactively.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		evalID := args[0]
		findingIdx, _ := cmd.Flags().GetInt("finding")
		execute, _ := cmd.Flags().GetBool("execute")

		if cfg.APIKey == "" {
			color.Red("❌ No API key configured. Run 'sofe upgrade' first.")
			return
		}

		green := color.New(color.FgGreen, color.Bold)
		yellow := color.New(color.FgYellow)

		// Get the evaluation with remediation commands
		cloudClient := getCloudClient()
		data, err := cloudClient.Get("/evaluations/" + evalID)
		if err != nil {
			color.Red("❌ %s", err)
			return
		}

		var eval struct {
			Findings []struct {
				PolicyName          string   `json:"policy_name"`
				ResourceID          string   `json:"resource_id"`
				Message             string   `json:"message"`
				RemediationCommands []string `json:"remediation_commands"`
			} `json:"findings"`
		}
		json.Unmarshal(data, &eval)

		if findingIdx < 0 || findingIdx >= len(eval.Findings) {
			color.Red("❌ Finding index %d out of range (0-%d)", findingIdx, len(eval.Findings)-1)
			return
		}

		finding := eval.Findings[findingIdx]
		green.Printf("🔧 Remediation for finding #%d\n", findingIdx)
		fmt.Printf("   Policy: %s\n", finding.PolicyName)
		fmt.Printf("   Resource: %s\n", finding.ResourceID)
		fmt.Println()

		if len(finding.RemediationCommands) == 0 {
			yellow.Println("⚠️  No remediation commands available for this finding.")
			fmt.Println("   This policy may require manual intervention.")
			return
		}

		fmt.Println("Commands:")
		for i, cmdStr := range finding.RemediationCommands {
			fmt.Printf("  %d. %s\n", i+1, cmdStr)
		}
		fmt.Println()

		if !execute {
			fmt.Println("Run with --execute to apply these commands interactively.")
			return
		}

		// Execute mode
		yellow.Println("⚡ Execute mode — each command requires confirmation")
		fmt.Println()
		reader := bufio.NewReader(os.Stdin)

		for i, cmdStr := range finding.RemediationCommands {
			fmt.Printf("[%d/%d] %s\n", i+1, len(finding.RemediationCommands), cmdStr)
			fmt.Print("  Execute? [y/N]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))

			if input != "y" && input != "yes" {
				yellow.Println("  ⏭ Skipped")
				continue
			}

			// Execute the command
			parts := strings.Fields(cmdStr)
			if len(parts) == 0 {
				continue
			}
			execCmd := exec.Command(parts[0], parts[1:]...)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr

			if err := execCmd.Run(); err != nil {
				color.Red("  ❌ Failed: %s", err)
			} else {
				green.Println("  ✅ Done")
			}
			fmt.Println()
		}

		green.Println("🎉 Remediation complete")
	},
}

func init() {
	remediateCmd.Flags().IntP("finding", "f", 0, "Finding index to remediate (0-based)")
	remediateCmd.Flags().Bool("execute", false, "Actually execute the commands (with confirmation)")
	rootCmd.AddCommand(remediateCmd)
}
