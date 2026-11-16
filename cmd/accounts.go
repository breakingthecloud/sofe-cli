package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Account administration for cloud mode (S116-B5.2).
//
// The SaaS platform cannot update a customer's CloudFormation stack: the SOFE role is read-only by
// design. This command closes that gap without expanding IAM: it lists the accounts connected to the
// user's account in api.sofe.dev, maps them to the AWS profiles available on this machine
// (aws configure list-profiles + sts get-caller-identity), and runs `aws cloudformation update-stack`
// locally with the user's own credentials.

const (
	defaultStackName      = "sofe-read-only"
	defaultStackRegion    = "us-east-1"
	templateV23Date       = "2026-11-16" // accounts connected before this date may still run v2.2
	cloudOnlyInstructions = "this command needs cloud mode: run `sofe login` or `sofe config set mode cloud` + `sofe config set api-key sk_sofe_...`"
)

type cloudAccount struct {
	ID          string `json:"id"`
	AccountID   string `json:"accountId"`
	Alias       string `json:"alias"`
	ConnectedAt string `json:"connectedAt"`
	Status      string `json:"status"`
}

type cloudAccountsResponse struct {
	Accounts []cloudAccount `json:"accounts"`
	Total    int            `json:"total"`
}

type cloudCfnConfig struct {
	ExternalID  string `json:"external_id"`
	TemplateURL string `json:"template_url"`
	StackName   string `json:"stack_name"`
}

var (
	accountsRegion     string
	accountsStackName  string
	accountsProfileMap string
	accountsDryRun     bool
	accountsYes        bool
	accountsAccount    string
	accountsListCheck  bool
	accountsListRegion string
	accountsListStack  string
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List and update the SOFE role stack in your connected AWS accounts (cloud mode)",
}

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the AWS accounts connected to your SOFE account",
	RunE:  runAccountsList,
}

var accountsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the SOFE role stack in every connected account",
	Long: `Update the SOFE read-only role stack in the AWS accounts connected to your SOFE account.

It uses your LOCAL AWS credentials (SSO/CLI). SOFE never sees them. The command maps each connected
account to a local profile by calling sts get-caller-identity for the profiles in ~/.aws/config, skips
stacks already on the current template, and verifies the TemplateVersion output afterwards.

Use --profile-map when the automatic mapping is not what you want:
  sofe accounts update --profile-map 111111111111=prod,222222222222=dev`,
	RunE: runAccountsUpdate,
}

func init() {
	accountsUpdateCmd.Flags().StringVar(&accountsRegion, "region", defaultStackRegion, "AWS region of the CloudFormation stack")
	accountsUpdateCmd.Flags().StringVar(&accountsStackName, "stack-name", "", "CloudFormation stack name (default: the one reported by the platform)")
	accountsUpdateCmd.Flags().StringVar(&accountsProfileMap, "profile-map", "", "Map account IDs to local AWS profiles: 111111111111=prod,222222222222=dev")
	accountsUpdateCmd.Flags().BoolVar(&accountsDryRun, "dry-run", false, "Print the commands without running them")
	accountsUpdateCmd.Flags().BoolVar(&accountsYes, "yes", false, "Do not ask for confirmation")
	accountsUpdateCmd.Flags().Bool("all", true, "Update every connected account (default; kept for discoverability)")
	accountsUpdateCmd.Flags().StringVar(&accountsAccount, "account", "", "Update only this AWS account ID")
	accountsListCmd.Flags().BoolVar(&accountsListCheck, "check", true, "Ask AWS for each stack's current TemplateVersion (uses local profiles)")
	accountsListCmd.Flags().StringVar(&accountsListRegion, "region", defaultStackRegion, "AWS region of the CloudFormation stack")
	accountsListCmd.Flags().StringVar(&accountsListStack, "stack-name", defaultStackName, "CloudFormation stack name")
	accountsCmd.AddCommand(accountsListCmd, accountsUpdateCmd)
	rootCmd.AddCommand(accountsCmd)
}

func requireCloudMode() error {
	if cfg == nil || cfg.APIKey == "" {
		return fmt.Errorf("%s", cloudOnlyInstructions)
	}
	return nil
}

func fetchCloudAccounts() ([]cloudAccount, error) {
	data, err := getCloudClient().Get("/connect/accounts")
	if err != nil {
		return nil, fmt.Errorf("could not list connected accounts: %w", err)
	}
	var parsed cloudAccountsResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("unexpected answer from the platform: %w", err)
	}
	return parsed.Accounts, nil
}

func fetchCloudCfn() (cloudCfnConfig, error) {
	data, err := getCloudClient().Get("/connect/cfn")
	if err != nil {
		return cloudCfnConfig{}, fmt.Errorf("could not read the CloudFormation template URL: %w", err)
	}
	var parsed cloudCfnConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		return cloudCfnConfig{}, fmt.Errorf("unexpected answer from the platform: %w", err)
	}
	if parsed.TemplateURL == "" {
		return parsed, fmt.Errorf("the platform did not return a template URL")
	}
	if parsed.StackName == "" {
		parsed.StackName = defaultStackName
	}
	return parsed, nil
}

func runAccountsList(_ *cobra.Command, _ []string) error {
	if err := requireCloudMode(); err != nil {
		return err
	}
	accounts, err := fetchCloudAccounts()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		fmt.Println("No accounts connected. Connect one at https://platform.sofe.dev/accounts.")
		return nil
	}
	profiles := map[string]string{}
	if accountsListCheck {
		profiles = discoverAccountProfiles()
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ALIAS\tACCOUNT\tCONNECTED\tTEMPLATE")
	for _, a := range accounts {
		state := "not checked"
		if accountsListCheck {
			if profile, ok := profiles[a.AccountID]; ok {
				if version := stackTemplateVersion(profile, accountsListRegion, accountsListStack); version != "" {
					state = "v" + version
				} else {
					state = "stack not found"
				}
			} else {
				state = "no local profile"
			}
		} else if mayNeedStackUpdate(a.ConnectedAt) {
			state = "likely outdated"
		} else {
			state = "current?"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Alias, a.AccountID, shortDate(a.ConnectedAt), state)
	}
	return w.Flush()
}

func runAccountsUpdate(_ *cobra.Command, _ []string) error {
	if err := requireCloudMode(); err != nil {
		return err
	}
	accounts, err := fetchCloudAccounts()
	if err != nil {
		return err
	}
	if accountsAccount != "" {
		var filtered []cloudAccount
		for _, a := range accounts {
			if a.AccountID == accountsAccount {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("account %s is not connected to your SOFE account", accountsAccount)
		}
		accounts = filtered
	}
	if len(accounts) == 0 {
		fmt.Println("No accounts connected. Nothing to update.")
		return nil
	}

	cfn, err := fetchCloudCfn()
	if err != nil {
		return err
	}
	stackName := accountsStackName
	if stackName == "" {
		stackName = cfn.StackName
	}
	region := accountsRegion
	if region == "" {
		region = defaultStackRegion
	}

	profiles := parseProfileMap(accountsProfileMap)
	// Read-only STS calls: do this even in --dry-run so the printed plan matches what would run.
	for account, profile := range discoverAccountProfiles() {
		if _, exists := profiles[account]; !exists {
			profiles[account] = profile
		}
	}

	type plan struct {
		account cloudAccount
		profile string
	}
	var plans []plan
	var missing []cloudAccount
	for _, a := range accounts {
		if p, ok := profiles[a.AccountID]; ok && p != "" {
			plans = append(plans, plan{account: a, profile: p})
		} else {
			missing = append(missing, a)
		}
	}

	if len(plans) == 0 {
		printMissingProfiles(missing, stackName, region, cfn.TemplateURL)
		return fmt.Errorf("no local AWS profile matched the connected accounts")
	}

	fmt.Println()
	fmt.Printf("Update plan (%d account(s), stack %q, region %s):\n", len(plans), stackName, region)
	for _, p := range plans {
		fmt.Printf("  • %s (%s) → profile %s\n", p.account.Alias, p.account.AccountID, p.profile)
	}
	if len(missing) > 0 {
		fmt.Printf("  • %d account(s) without a matching local profile will be skipped\n", len(missing))
	}
	fmt.Println()

	if accountsDryRun {
		for _, p := range plans {
			fmt.Println(awsShellCommand(updateStackArgs(p.profile, region, stackName, cfn.TemplateURL)))
		}
		printMissingProfiles(missing, stackName, region, cfn.TemplateURL)
		return nil
	}

	if !accountsYes && !confirm("Continue? [y/N] ") {
		fmt.Println("Cancelled.")
		return nil
	}

	failed := 0
	for _, p := range plans {
		before := stackTemplateVersion(p.profile, region, stackName)
		out, runErr := runAws(updateStackArgs(p.profile, region, stackName, cfn.TemplateURL)...)
		if runErr != nil {
			if strings.Contains(strings.ToLower(out), "no updates are to be performed") {
				fmt.Printf("✓ %s (%s): already current (TemplateVersion %s)\n", p.account.Alias, p.account.AccountID, orDash(before))
				continue
			}
			failed++
			fmt.Printf("✗ %s (%s): %s\n", p.account.Alias, p.account.AccountID, firstLine(out))
			continue
		}
		if _, waitErr := runAws("cloudformation", "wait", "stack-update-complete", "--stack-name", stackName, "--region", region, "--profile", p.profile); waitErr != nil {
			failed++
			fmt.Printf("✗ %s (%s): update did not complete: %s\n", p.account.Alias, p.account.AccountID, firstLine(out))
			continue
		}
		after := stackTemplateVersion(p.profile, region, stackName)
		fmt.Printf("✓ %s (%s): TemplateVersion %s → %s\n", p.account.Alias, p.account.AccountID, orDash(before), orDash(after))
	}

	printMissingProfiles(missing, stackName, region, cfn.TemplateURL)
	if failed > 0 {
		return fmt.Errorf("%d stack update(s) failed", failed)
	}
	return nil
}

func printMissingProfiles(missing []cloudAccount, stackName, region, templateUrl string) {
	if len(missing) == 0 {
		return
	}
	label := color.New(color.FgHiBlack)
	fmt.Println()
	fmt.Printf("%d account(s) have no local AWS profile that can administer them:\n", len(missing))
	for _, a := range missing {
		fmt.Printf("  • %s (%s)\n", a.Alias, a.AccountID)
		fmt.Println(label.Sprintf("    %s", awsShellCommand(updateStackArgs("YOUR_PROFILE", region, stackName, templateUrl))))
	}
	fmt.Println(label.Sprint("  Or update that stack in the CloudFormation console (Update stack → Replace current template)."))
}

func mayNeedStackUpdate(connectedAt string) bool {
	t, err := time.Parse(time.RFC3339, connectedAt)
	if err != nil {
		return false
	}
	cutoff, err := time.Parse("2006-01-02", templateV23Date)
	if err != nil {
		return false
	}
	return t.Before(cutoff)
}

func shortDate(iso string) string {
	if t, err := time.Parse(time.RFC3339, iso); err == nil {
		return t.Format("2006-01-02")
	}
	return iso
}

func parseProfileMap(value string) map[string]string {
	out := map[string]string{}
	for _, pair := range strings.Split(value, ",") {
		account, profile, ok := strings.Cut(strings.TrimSpace(pair), "=")
		account, profile = strings.TrimSpace(account), strings.TrimSpace(profile)
		if ok && validAccountID(account) && profile != "" {
			out[account] = profile
		}
	}
	return out
}

// discoverAccountProfiles maps account IDs to local AWS profiles. It ignores profiles whose SSO session
// is expired or that belong to another account, so a stale profile never updates the wrong stack.
func discoverAccountProfiles() map[string]string {
	candidates := []string{cfg.AWSProfile}
	out, err := exec.Command("aws", "configure", "list-profiles").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if profile := strings.TrimSpace(line); profile != "" {
				candidates = append(candidates, profile)
			}
		}
	}
	seen := map[string]bool{}
	var profiles []string
	for _, profile := range candidates {
		if profile != "" && !seen[profile] {
			seen[profile] = true
			profiles = append(profiles, profile)
		}
	}
	return profileMapFromCandidates(profiles, func(profile string) (string, bool) {
		out, err := exec.Command(
			"aws", "sts", "get-caller-identity",
			"--profile", profile, "--query", "Account", "--output", "text",
		).Output()
		if err != nil {
			return "", false
		}
		account := strings.TrimSpace(string(out))
		return account, validAccountID(account)
	})
}

func profileMapFromCandidates(candidates []string, accountFor func(string) (string, bool)) map[string]string {
	out := map[string]string{}
	for _, profile := range candidates {
		if account, ok := accountFor(profile); ok && account != "" {
			out[account] = profile
		}
	}
	return out
}

func updateStackArgs(profile, region, stackName, templateUrl string) []string {
	args := []string{
		"cloudformation", "update-stack",
		"--stack-name", stackName,
		"--template-url", templateUrl,
		"--parameters", "ParameterKey=ExternalId,UsePreviousValue=true",
		"--capabilities", "CAPABILITY_NAMED_IAM",
		"--region", region,
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	return args
}

func stackTemplateVersion(profile, region, stackName string) string {
	args := []string{
		"cloudformation", "describe-stacks",
		"--stack-name", stackName,
		"--region", region,
		"--query", "Stacks[0].Outputs[?OutputKey=='TemplateVersion'].OutputValue | [0]",
		"--output", "text",
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}
	out, err := exec.Command("aws", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runAws(args ...string) (string, error) {
	out, err := exec.Command("aws", args...).CombinedOutput()
	return string(out), err
}

func awsShellCommand(args []string) string {
	return "aws " + shellCommand(args)
}

func shellCommand(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\"'") {
			quoted = append(quoted, fmt.Sprintf("%q", arg))
		} else {
			quoted = append(quoted, arg)
		}
	}
	return strings.Join(quoted, " ")
}

func confirm(prompt string) bool {
	fmt.Print(prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if i := strings.IndexByte(value, '\n'); i >= 0 {
		return value[:i]
	}
	if value == "" {
		return "aws command failed"
	}
	return value
}

func orDash(value string) string {
	if value == "" {
		return "?"
	}
	return value
}
