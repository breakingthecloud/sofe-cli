package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var connectAwsCmd = &cobra.Command{
	Use:   "connect-aws",
	Short: "Create/update the SOFE read-only role in your AWS account",
	Long:  "Uses your LOCAL AWS credentials (aws CLI / SSO) to deploy the SOFE CloudFormation template. SOFE never sees your credentials.",
	Run: func(cmd *cobra.Command, args []string) {
		if cfg.APIKey == "" {
			color.Yellow("No API key. Run 'sofe login' or 'sofe upgrade' first.")
			return
		}

		label := color.New(color.FgHiBlack)
		green := color.New(color.FgGreen, color.Bold)
		red := color.New(color.FgRed)

		fmt.Println()
		fmt.Println(color.New(color.FgGreen, color.Bold).Sprint("SOFE Connect AWS"))
		fmt.Println(label.Sprint("  Deploying the SOFE read-only role using your local AWS credentials..."))
		fmt.Println()

		cloudClient := getCloudClient()
		data, err := cloudClient.Get("/connect/cfn")
		if err != nil {
			red.Printf("❌ Failed to fetch connection info: %s\n", err)
			return
		}

		var cfn struct {
			ExternalID  string `json:"external_id"`
			TemplateURL string `json:"template_url"`
			StackName   string `json:"stack_name"`
		}
		if err := json.Unmarshal(data, &cfn); err != nil {
			red.Printf("❌ Bad response from API: %s\n", err)
			return
		}
		if cfn.ExternalID == "" {
			color.Yellow("No external ID yet — visit platform.sofe.dev/accounts to create your connection first.")
			return
		}

		// Download the template locally, then `aws cloudformation deploy --template-file`
		// (deploy does NOT support --template-url; it manages create/update idempotently).
		tmpFile, err := os.CreateTemp("", "sofe-role-*.yaml")
		if err != nil {
			red.Printf("❌ Cannot create temp file: %s\n", err)
			return
		}
		defer os.Remove(tmpFile.Name())

		resp, err := http.Get(cfn.TemplateURL)
		if err != nil {
			red.Printf("❌ Cannot download template: %s\n", err)
			return
		}
		defer resp.Body.Close()
		if _, err := io.Copy(tmpFile, resp.Body); err != nil {
			red.Printf("❌ Cannot write template: %s\n", err)
			return
		}
		tmpFile.Close()

		awsArgs := []string{
			"cloudformation", "deploy",
			"--template-file", tmpFile.Name(),
			"--stack-name", cfn.StackName,
			"--parameter-overrides", "ExternalId=" + cfn.ExternalID,
			"--capabilities", "CAPABILITY_NAMED_IAM",
		}
		fmt.Println("☁️  aws cloudformation deploy --stack-name " + cfn.StackName)
		out, err := exec.Command("aws", awsArgs...).CombinedOutput()
		if err != nil {
			red.Printf("❌ aws cloudformation failed: %s\n", err)
			fmt.Println(string(out))
			color.Yellow("  Ensure you're logged in: aws sso login --profile <your-profile> (or set AWS_PROFILE).")
			return
		}
		fmt.Println(string(out))
		green.Println("✅ SOFE read-only role deployed/updated: " + cfn.StackName)
		fmt.Println("  Next: copy the role ARN (from Outputs) → platform.sofe.dev/accounts → Connect.")
		fmt.Println("  Docs: " + cfn.TemplateURL)
	},
}

func init() {
	rootCmd.AddCommand(connectAwsCmd)
}