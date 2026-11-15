package cmd

import (
	"strings"
	"testing"

	"github.com/breakingthecloud/sofe-cli/internal/client"
)

func f64(v float64) *float64 { return &v }

func TestValidAccountID(t *testing.T) {
	for id, want := range map[string]bool{
		"111111111111":                   true,
		"000000000000":                   true,
		"11111111111":                    false, // 11 digits
		"1111111111111":                  false, // 13 digits
		"11111111111a":                   false,
		"":                               false,
		" 111111111111":                  false,
		"1111 1111 1111":                 false,
		"arn:aws:iam::111111111111:root": false,
	} {
		if got := validAccountID(id); got != want {
			t.Errorf("validAccountID(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestDescribeAccountRequiredListsEveryAccount(t *testing.T) {
	text := describeCloudError(&client.APIError{
		Status: 400, Code: "account_required",
		Accounts: []client.AccountRef{{AccountID: "111111111111", Alias: "dev"}, {AccountID: "222222222222", Alias: "payer"}},
	})
	for _, want := range []string{"Choose one with --account", "--account 111111111111   dev", "--account 222222222222   payer"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}

func TestDescribeAccountRequiredWithoutAList(t *testing.T) {
	text := describeCloudError(&client.APIError{Status: 400, Code: "account_required"})
	if !strings.Contains(text, "platform.sofe.dev/accounts") {
		t.Errorf("should point to the accounts page:\n%s", text)
	}
}

func TestDescribeAccountWithoutAlias(t *testing.T) {
	text := describeCloudError(&client.APIError{
		Status: 400, Code: "account_required", Accounts: []client.AccountRef{{AccountID: "111111111111"}},
	})
	if !strings.Contains(text, "--account 111111111111") || strings.Contains(text, "111111111111   ") {
		t.Errorf("no trailing alias column expected:\n%q", text)
	}
}

func TestDescribeOtherErrors(t *testing.T) {
	cases := []struct {
		name string
		err  client.APIError
		want []string
	}{
		{"not connected", client.APIError{Status: 404, Code: "account_not_connected", Message: "AWS account 111111111111 is not connected to your SOFE account."},
			[]string{"111111111111 is not connected", "platform.sofe.dev/accounts"}},
		{"role denied", client.APIError{Status: 403, Code: "role_assume_denied", Message: "Could not assume the role."},
			[]string{"Could not assume the role.", "External ID"}},
		{"rate limit", client.APIError{Status: 429, Code: "Rate limit exceeded (10/day free tier). Upgrade at platform.sofe.dev"},
			[]string{"Daily evaluation limit", "00:00 UTC", "platform.sofe.dev/pricing"}},
		{"credits", client.APIError{Status: 429, Code: "Credits exhausted", Message: "Your monthly credits are used up."},
			[]string{"monthly credits are used up", "platform.sofe.dev/pricing"}},
	}
	for _, c := range cases {
		e := c.err
		text := describeCloudError(&e)
		for _, want := range c.want {
			if !strings.Contains(text, want) {
				t.Errorf("%s: missing %q in %q", c.name, want, text)
			}
		}
	}
}

func TestDescribeUnknownErrorKeepsTheOriginalMessage(t *testing.T) {
	e := &client.APIError{Status: 502, Body: `{"error":"evaluation_backend_error"}`}
	if got := describeCloudError(e); got != e.Error() {
		t.Errorf("got %q, want %q", got, e.Error())
	}
}

func TestAccountLine(t *testing.T) {
	for _, c := range []struct {
		resp client.EvaluateResponse
		want string
	}{
		{client.EvaluateResponse{AccountID: "111111111111", AccountAlias: "dev"}, "dev (111111111111)"},
		{client.EvaluateResponse{AccountID: "111111111111"}, "111111111111"},
		{client.EvaluateResponse{}, ""},
	} {
		r := c.resp
		if got := accountLine(&r); got != c.want {
			t.Errorf("accountLine = %q, want %q", got, c.want)
		}
	}
}

func TestCostLine(t *testing.T) {
	partial := client.EvaluateResponse{CostStatus: "partial", CostAccountMonthly: f64(40.00), CostMatchedMonthly: f64(1.2), CostCoverageUSD: f64(0.03)}
	if got, want := costLine(&partial), "$40.00/mo (attributed to resources: $1.20, 3%)"; got != want {
		t.Errorf("partial = %q, want %q", got, want)
	}
	tiny := client.EvaluateResponse{CostStatus: "partial", CostAccountMonthly: f64(500), CostMatchedMonthly: f64(1), CostCoverageUSD: f64(0.002)}
	if got := costLine(&tiny); !strings.Contains(got, "<1%") {
		t.Errorf("a tiny share must not read 0%%: %q", got)
	}
	noRatio := client.EvaluateResponse{CostStatus: "ok", CostAccountMonthly: f64(100), CostMatchedMonthly: f64(98)}
	if got, want := costLine(&noRatio), "$100.00/mo (attributed to resources: $98.00)"; got != want {
		t.Errorf("ok without ratio = %q, want %q", got, want)
	}
}

func TestCostLineNeverInventsNumbers(t *testing.T) {
	for _, status := range []string{"", "unknown", "something_new"} {
		if got := costLine(&client.EvaluateResponse{CostStatus: status}); got != "" {
			t.Errorf("status %q printed %q, want nothing", status, got)
		}
	}
	// ok/partial without numbers says nothing instead of "$0.00".
	if got := costLine(&client.EvaluateResponse{CostStatus: "partial"}); got != "" {
		t.Errorf("partial without numbers printed %q", got)
	}
	for status, want := range map[string]string{
		"access_denied": "template v2.1",
		"not_enabled":   "Cost Explorer",
		"no_data":       "no spend",
		"error":         "temporarily unavailable",
	} {
		if got := costLine(&client.EvaluateResponse{CostStatus: status}); !strings.Contains(got, want) {
			t.Errorf("status %s: %q should contain %q", status, got, want)
		}
	}
}

func TestSummaryCardShowsAccountAndCost(t *testing.T) {
	resp := &client.EvaluateResponse{
		EvaluationID: "11111111-2222-3333-4444-555555555555", FindingsCount: 3,
		AccountID: "111111111111", AccountAlias: "dev",
		CostStatus: "partial", CostAccountMonthly: f64(40.00), CostMatchedMonthly: f64(1.2), CostCoverageUSD: f64(0.03),
	}
	card := renderSummaryCard(resp)
	for _, want := range []string{"Account:", "dev (111111111111)", "Cost:", "$40.00/mo"} {
		if !strings.Contains(card, want) {
			t.Errorf("summary card missing %q:\n%s", want, card)
		}
	}
	// An older server (no account, no cost) keeps the card as before.
	old := renderSummaryCard(&client.EvaluateResponse{EvaluationID: "11111111-2222-3333-4444-555555555555"})
	if strings.Contains(old, "Account:") || strings.Contains(old, "Cost:") {
		t.Errorf("old-server card must not show Account/Cost:\n%s", old)
	}
}
