package cmd

import (
	"fmt"
	"os"

	"github.com/breakingthecloud/sofe-cli/internal/output"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var policiesCmd = &cobra.Command{
	Use:   "policies",
	Short: "List loaded policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := policiesDir
		if dir == "" {
			dir = cfg.PoliciesDir
		}

		policies, err := apiClient.Policies(dir)
		if err != nil {
			output.PrintError(err.Error())
			return err
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Severity", "Resource Types", "Metric"})
		table.SetBorder(false)

		for _, p := range policies {
			types := ""
			for i, t := range p.ResourceTypes {
				if i > 0 {
					types += ", "
				}
				types += t
			}
			table.Append([]string{p.Name, p.Severity, types, p.Metric})
		}

		table.Render()
		fmt.Printf("\n%d policies loaded\n", len(policies))
		return nil
	},
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check SOFE server health",
	RunE: func(cmd *cobra.Command, args []string) error {
		h, err := apiClient.Health()
		if err != nil {
			output.PrintError(err.Error())
			return err
		}
		output.PrintSuccess(fmt.Sprintf("SOFE Server %s (%s)", h.Version, h.Status))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(policiesCmd)
	rootCmd.AddCommand(healthCmd)
}
