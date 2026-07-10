package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var terraformCmd = &cobra.Command{
	Use:   "terraform <path>",
	Short: "Scan Terraform plans for FinOps issues (coming soon)",
	Long:  "Pre-deploy scan of .tf files or tfplan.json for cost and architecture policy violations.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		yellow := color.New(color.FgYellow, color.Bold)
		yellow.Println("🚧 sofe terraform — Coming Soon (SoW-S047)")
		fmt.Println()
		fmt.Println("This command will:")
		fmt.Println("  • Parse Terraform plan JSON (terraform show -json tfplan)")
		fmt.Println("  • Evaluate FinOps policies against planned resources")
		fmt.Println("  • Report findings BEFORE you apply")
		fmt.Println("  • Block deploys that violate critical policies")
		fmt.Println()
		fmt.Printf("  Path provided: %s\n", args[0])
		fmt.Println()
		fmt.Println("Track progress: https://github.com/breakingthecloud/sofe-cli/issues")
		fmt.Println("Expected in Sprint 26.")
	},
}

func init() {
	rootCmd.AddCommand(terraformCmd)
}
