package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/breakingthecloud/sofe-cli/internal/client"
)

// Helpers for `sofe evaluate --cloud --account <id>` (S111 Phase D). Pure functions, so they are unit tested.

var accountIDPattern = regexp.MustCompile(`^[0-9]{12}$`)

// validAccountID reports whether id looks like an AWS account ID (12 digits).
func validAccountID(id string) bool {
	return accountIDPattern.MatchString(id)
}

func apiErrorText(e *client.APIError) string {
	switch {
	case e.Message != "":
		return e.Message
	case e.Code != "":
		return e.Code
	default:
		return e.Body
	}
}

// describeCloudError turns an error answer of api.sofe.dev into text that says what to do next.
func describeCloudError(e *client.APIError) string {
	switch {
	case e.Status == 400 && e.Code == "account_required":
		var b strings.Builder
		b.WriteString("Several AWS accounts are connected. Choose one with --account:\n")
		for _, a := range e.Accounts {
			fmt.Fprintf(&b, "  --account %s", a.AccountID)
			if a.Alias != "" {
				fmt.Fprintf(&b, "   %s", a.Alias)
			}
			b.WriteString("\n")
		}
		if len(e.Accounts) == 0 {
			b.WriteString("  (your connected accounts are listed at platform.sofe.dev/accounts)\n")
		}
		return strings.TrimRight(b.String(), "\n")
	case e.Status == 404 && e.Code == "account_not_connected":
		return apiErrorText(e) + "\nYour connected accounts are listed at platform.sofe.dev/accounts"
	case e.Status == 403 && e.Code == "role_assume_denied":
		return apiErrorText(e) + "\nCheck the CloudFormation stack and its External ID at platform.sofe.dev/accounts"
	case e.Status == 429 && e.Code == "Credits exhausted":
		return apiErrorText(e) + "\nUpgrade at platform.sofe.dev/pricing"
	case e.Status == 429:
		return "Daily evaluation limit of your plan reached. It resets at 00:00 UTC.\nUpgrade at platform.sofe.dev/pricing"
	default:
		return e.Error()
	}
}

// accountLine says which account was evaluated; "" when the server did not say.
func accountLine(resp *client.EvaluateResponse) string {
	switch {
	case resp.AccountID == "":
		return ""
	case resp.AccountAlias != "":
		return fmt.Sprintf("%s (%s)", resp.AccountAlias, resp.AccountID)
	default:
		return resp.AccountID
	}
}

func pctText(ratio float64) string {
	pct := ratio * 100
	if pct > 0 && pct < 1 {
		return "<1%"
	}
	return fmt.Sprintf("%.0f%%", pct)
}

// costLine summarizes the cost data of the evaluation; "" when there is nothing honest to say
// (an older server, or no status). An unknown cost is never printed as $0.
func costLine(resp *client.EvaluateResponse) string {
	switch resp.CostStatus {
	case "ok", "partial":
		if resp.CostAccountMonthly == nil || resp.CostMatchedMonthly == nil {
			return ""
		}
		attributed := fmt.Sprintf("$%.2f", *resp.CostMatchedMonthly)
		if resp.CostCoverageUSD != nil {
			attributed += ", " + pctText(*resp.CostCoverageUSD)
		}
		return fmt.Sprintf("$%.2f/mo (attributed to resources: %s)", *resp.CostAccountMonthly, attributed)
	case "access_denied":
		return "not available: update the CloudFormation stack to template v2.1"
	case "not_enabled":
		return "not available: enable resource-level data in AWS Cost Explorer"
	case "no_data":
		return "no spend in the last 7 days"
	case "error":
		return "temporarily unavailable"
	default:
		return ""
	}
}
