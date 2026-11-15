package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeAPI answers every request with status/body and records what was sent.
func fakeAPI(t *testing.T, status int, body string) (*Client, *map[string]any) {
	t.Helper()
	sent := map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &sent)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return New(srv.URL, "sk_sofe_test"), &sent
}

func TestEvaluateSendsAccountIDWhenGiven(t *testing.T) {
	c, sent := fakeAPI(t, 200, `{"evaluation_id":"e1"}`)
	if _, err := c.Evaluate(EvaluateRequest{PoliciesDir: "p", AccountID: "916272138274"}); err != nil {
		t.Fatal(err)
	}
	if got := (*sent)["account_id"]; got != "916272138274" {
		t.Fatalf("account_id sent = %v, want 916272138274", got)
	}
}

func TestEvaluateOmitsAccountIDWhenEmpty(t *testing.T) {
	// The local server must not receive a field it doesn't know, and a single-account cloud user must not need it.
	c, sent := fakeAPI(t, 200, `{"evaluation_id":"e1"}`)
	if _, err := c.Evaluate(EvaluateRequest{PoliciesDir: "p"}); err != nil {
		t.Fatal(err)
	}
	if _, present := (*sent)["account_id"]; present {
		t.Fatalf("account_id must be omitted when empty, request had %v", *sent)
	}
}

func TestAccountRequiredIsATypedErrorWithTheList(t *testing.T) {
	body := `{"error":"account_required","message":"Several AWS accounts are connected.","accounts":[` +
		`{"account_id":"916272138274","alias":"dev"},{"account_id":"279624932954","alias":"payer"}]}`
	c, _ := fakeAPI(t, 400, body)
	_, err := c.Evaluate(EvaluateRequest{PoliciesDir: "p"})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T: %v", err, err)
	}
	if apiErr.Status != 400 || apiErr.Code != "account_required" {
		t.Fatalf("status/code = %d/%q", apiErr.Status, apiErr.Code)
	}
	if len(apiErr.Accounts) != 2 || apiErr.Accounts[1].AccountID != "279624932954" || apiErr.Accounts[1].Alias != "payer" {
		t.Fatalf("accounts = %+v", apiErr.Accounts)
	}
}

func TestAPIErrorKeepsTheOriginalMessageFormat(t *testing.T) {
	c, _ := fakeAPI(t, 429, `{"error":"Rate limit exceeded"}`)
	_, err := c.Evaluate(EvaluateRequest{PoliciesDir: "p"})
	want := `API error 429: {"error":"Rate limit exceeded"}`
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestAPIErrorWithANonJSONBody(t *testing.T) {
	// An HTML error page from a gateway: no panic, the body is kept, the fields stay empty.
	c, _ := fakeAPI(t, 502, `<html>Bad Gateway</html>`)
	_, err := c.Evaluate(EvaluateRequest{PoliciesDir: "p"})

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T", err)
	}
	if apiErr.Code != "" || apiErr.Message != "" || len(apiErr.Accounts) != 0 {
		t.Fatalf("fields should be empty: %+v", apiErr)
	}
	if !strings.Contains(apiErr.Body, "Bad Gateway") {
		t.Fatalf("body lost: %q", apiErr.Body)
	}
}

func TestResponseCarriesAccountAndCostFields(t *testing.T) {
	body := `{"evaluation_id":"e1","findings_count":3,"account_id":"916272138274","account_alias":"dev",` +
		`"cost_status":"partial","cost_coverage_usd":0.0297,"cost_account_monthly":26.92,"cost_matched_monthly":0.8}`
	c, _ := fakeAPI(t, 200, body)
	r, err := c.Evaluate(EvaluateRequest{PoliciesDir: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if r.AccountID != "916272138274" || r.AccountAlias != "dev" || r.CostStatus != "partial" {
		t.Fatalf("account/cost status = %q/%q/%q", r.AccountID, r.AccountAlias, r.CostStatus)
	}
	if r.CostAccountMonthly == nil || *r.CostAccountMonthly != 26.92 || r.CostMatchedMonthly == nil || *r.CostMatchedMonthly != 0.8 {
		t.Fatalf("cost numbers = %v / %v", r.CostAccountMonthly, r.CostMatchedMonthly)
	}
}

func TestOlderServerWithoutCostFieldsLeavesThemUnknown(t *testing.T) {
	// "No data" must stay nil, never 0.
	c, _ := fakeAPI(t, 200, `{"evaluation_id":"e1","findings_count":3}`)
	r, err := c.Evaluate(EvaluateRequest{PoliciesDir: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if r.CostStatus != "" || r.CostAccountMonthly != nil || r.CostMatchedMonthly != nil || r.CostCoverageUSD != nil {
		t.Fatalf("cost fields should be empty: %+v", r)
	}
}

func TestJSONOutputOmitsUnknownCostFields(t *testing.T) {
	// `sofe evaluate --format json` prints this struct: older servers must not produce "cost_*": 0.
	raw, err := json.Marshal(&EvaluateResponse{EvaluationID: "e1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "cost_") {
		t.Fatalf("unknown cost leaked into JSON: %s", raw)
	}
}
