package cmd

import (
	"fmt"
	"os"

	"github.com/breakingthecloud/sofe-cli/internal/client"
	"github.com/breakingthecloud/sofe-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	policiesDir   string
	profile       string
	failOn        string
	resourceTypes []string
	autoServe     bool
)

var evaluateCmd = &cobra.Command{
	Use:   "evaluate",
	Short: "Evaluate policies against live AWS resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := policiesDir
		if dir == "" {
			dir = cfg.PoliciesDir
		}
		prof := profile
		if prof == "" {
			prof = cfg.AWSProfile
		}

		// Auto-serve: start server if not running
		shouldStop := false
		if autoServe {
			shouldStop = AutoServe(cfg.APIURL)
		}
		defer func() {
			if shouldStop {
				fmt.Println("\n⏹ Stopping auto-started server...")
				stopServer()
			}
		}()

		fmt.Printf("📋 Evaluating policies from: %s\n", dir)

		resp, err := apiClient.Evaluate(client.EvaluateRequest{
			PoliciesDir:   dir,
			Profile:       prof,
			ResourceTypes: resourceTypes,
			FailOn:        failOn,
		})
		if err != nil {
			output.PrintError(err.Error())
			return err
		}

		fmt.Printf("☁️  %d resources scanned | ⚡ %d findings\n\n", resp.ResourcesScanned, resp.FindingsCount)

		format := formatFlag
		if format == "" {
			format = cfg.DefaultFormat
		}

		switch format {
		case "json":
			output.PrintJSON(resp)
		case "markdown":
			output.PrintMarkdown(resp)
		default:
			output.PrintTable(resp)
		}

		if resp.Failed {
			fmt.Printf("\n❌ FAILED: findings at or above '%s' severity\n", failOn)
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	evaluateCmd.Flags().StringVarP(&policiesDir, "policies", "p", "", "Policies directory")
	evaluateCmd.Flags().StringVar(&profile, "profile", "", "AWS profile")
	evaluateCmd.Flags().StringVar(&failOn, "fail-on", "", "Fail if findings at/above severity (critical|high|medium|low)")
	evaluateCmd.Flags().StringSliceVar(&resourceTypes, "resource-types", nil, "Filter resource types")
	evaluateCmd.Flags().BoolVar(&autoServe, "auto-serve", false, "Auto-start server if not running (stops after evaluation)")
	rootCmd.AddCommand(evaluateCmd)
}
