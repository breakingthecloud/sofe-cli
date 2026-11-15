package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/breakingthecloud/sofe-cli/internal/client"
	"github.com/breakingthecloud/sofe-cli/internal/config"
)

const (
	acctDev   = "111111111111"
	acctPayer = "222222222222"
)

func ev(id, account, alias string) Evaluation {
	return Evaluation{ID: id, AccountID: account, AccountAlias: alias}
}

func ids(evals []Evaluation) string {
	out := make([]string, len(evals))
	for i, e := range evals {
		out[i] = e.ID
	}
	return strings.Join(out, ",")
}

func TestOneAccountIsLeftAlone(t *testing.T) {
	in := []Evaluation{ev("e3", acctDev, "dev"), ev("e2", acctDev, "dev"), ev("e1", "", "")} // e1: before account tracking
	got, err := scopeEvaluations(in, "")
	if err != nil {
		t.Fatal(err)
	}
	if ids(got.Evaluations) != "e3,e2,e1" || got.Note != "" {
		t.Fatalf("one account must not change anything: %q note=%q", ids(got.Evaluations), got.Note)
	}
}

func TestOnlyLegacyEvaluationsAreLeftAlone(t *testing.T) {
	got, err := scopeEvaluations([]Evaluation{ev("e2", "", ""), ev("e1", "", "")}, "")
	if err != nil || ids(got.Evaluations) != "e2,e1" || got.Note != "" {
		t.Fatalf("legacy-only list changed: %v %q %q", err, ids(got.Evaluations), got.Note)
	}
}

func TestSeveralAccountsUseTheMostRecentOneAndSayWhatElseThereIs(t *testing.T) {
	in := []Evaluation{ev("p2", acctPayer, "payer"), ev("d2", acctDev, "dev"), ev("p1", acctPayer, "payer"), ev("d1", acctDev, "dev")}
	got, err := scopeEvaluations(in, "")
	if err != nil {
		t.Fatal(err)
	}
	if ids(got.Evaluations) != "p2,p1" {
		t.Fatalf("want the newest evaluation's account (payer) only, got %s", ids(got.Evaluations))
	}
	for _, want := range []string{"payer (" + acctPayer + ")", "the most recent", "dev (" + acctDev + ")", "--account"} {
		if !strings.Contains(got.Note, want) {
			t.Errorf("note missing %q: %s", want, got.Note)
		}
	}
}

func TestEvaluationsWithoutAccountAreLeftOutAndCountedWhenSeveralAccounts(t *testing.T) {
	in := []Evaluation{ev("d3", acctDev, "dev"), ev("old2", "", ""), ev("p1", acctPayer, "payer"), ev("old1", "", "")}
	got, _ := scopeEvaluations(in, "")
	if ids(got.Evaluations) != "d3" {
		t.Fatalf("got %s", ids(got.Evaluations))
	}
	if !strings.Contains(got.Note, "2 older evaluation(s) without an account were left out") {
		t.Fatalf("legacy count missing: %s", got.Note)
	}
}

func TestAccountWithoutAliasIsShownByID(t *testing.T) {
	got, _ := scopeEvaluations([]Evaluation{ev("a", acctDev, ""), ev("b", acctPayer, "")}, "")
	if !strings.Contains(got.Note, acctDev) || !strings.Contains(got.Note, acctPayer) || strings.Contains(got.Note, "()") {
		t.Fatalf("note: %s", got.Note)
	}
}

func TestAccountFlagKeepsOnlyThatAccount(t *testing.T) {
	in := []Evaluation{ev("p2", acctPayer, "payer"), ev("d2", acctDev, "dev"), ev("old", "", ""), ev("d1", acctDev, "dev")}
	got, err := scopeEvaluations(in, acctDev)
	if err != nil {
		t.Fatal(err)
	}
	if ids(got.Evaluations) != "d2,d1" || got.Note != "" {
		t.Fatalf("got %s note=%q", ids(got.Evaluations), got.Note)
	}
}

func TestAccountFlagErrors(t *testing.T) {
	in := []Evaluation{ev("d1", acctDev, "dev")}
	if _, err := scopeEvaluations(in, "12345"); err == nil || !strings.Contains(err.Error(), "12-digit") {
		t.Errorf("bad id: %v", err)
	}
	if _, err := scopeEvaluations(in, acctPayer); err == nil || !strings.Contains(err.Error(), acctPayer) {
		t.Errorf("no match: %v", err)
	}
}

func TestAccountLabel(t *testing.T) {
	for _, c := range []struct{ id, alias, want string }{
		{acctDev, "dev", "dev (" + acctDev + ")"}, {acctDev, "", acctDev}, {"", "ghost", ""}, {"", "", ""},
	} {
		if got := accountDisplay(c.id, c.alias); got != c.want {
			t.Errorf("accountDisplay(%q,%q) = %q, want %q", c.id, c.alias, got, c.want)
		}
	}
}

// --- fetching ---------------------------------------------------------------------------------

func listServer(t *testing.T, body string) (*client.Client, *[]string) {
	t.Helper()
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return client.New(srv.URL, "sk_sofe_test"), &paths
}

const mixedList = `{"evaluations":[` +
	`{"id":"p2","account_id":"222222222222","account_alias":"payer","findings_count":9},` +
	`{"id":"d2","account_id":"111111111111","account_alias":"dev","findings_count":3},` +
	`{"id":"p1","account_id":"222222222222","account_alias":"payer","findings_count":8}]}`

func TestFetchAsksForEnoughRowsAndTruncatesAfterScoping(t *testing.T) {
	c, paths := listServer(t, mixedList)
	got, err := fetchScopedEvaluations(c, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if (*paths)[0] != "/evaluations?limit=50" {
		t.Fatalf("request = %s (must over-fetch to scope on the client)", (*paths)[0])
	}
	if ids(got.Evaluations) != "p2" || got.Note == "" {
		t.Fatalf("got %s note=%q", ids(got.Evaluations), got.Note)
	}
}

func TestFetchWithAccountFlag(t *testing.T) {
	c, _ := listServer(t, mixedList)
	got, err := fetchScopedEvaluations(c, 7, acctDev)
	if err != nil || ids(got.Evaluations) != "d2" {
		t.Fatalf("%v %s", err, ids(got.Evaluations))
	}
}

func TestFetchEmptyHistoryIsNotAnError(t *testing.T) {
	c, _ := listServer(t, `{"evaluations":[]}`)
	got, err := fetchScopedEvaluations(c, 7, "")
	if err != nil || len(got.Evaluations) != 0 {
		t.Fatalf("%v %v", err, got)
	}
}

func TestFetchRejectsABadAccountBeforeAnyRequest(t *testing.T) {
	c, paths := listServer(t, mixedList)
	if _, err := fetchScopedEvaluations(c, 7, "nope"); err == nil {
		t.Fatal("want an error")
	}
	if len(*paths) != 0 {
		t.Fatalf("no request expected, got %v", *paths)
	}
}

func TestFetchOnNonJSONIsAnError(t *testing.T) {
	c, _ := listServer(t, `<html>Bad Gateway</html>`)
	if _, err := fetchScopedEvaluations(c, 7, ""); err == nil {
		t.Fatal("a gateway page must not look like an empty history")
	}
}

// --- self-hosted / local mode ----------------------------------------------------------------------

// A local (self-hosted) evaluation uses the AWS profile / credentials of the machine that runs sofe-server and
// must never send an account_id: that field only exists in cloud mode.
func TestLocalEvaluateSendsTheProfileAndNeverAnAccountID(t *testing.T) {
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &sent)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"evaluation_id":"e1","findings_count":0,"failed":false,"findings":[]}`))
	}))
	defer srv.Close()

	oldClient, oldAccount, oldAuto, oldCfg := apiClient, accountFlag, autoServe, cfg
	apiClient, accountFlag, autoServe, cfg = client.New(srv.URL, ""), acctDev, false, config.DefaultConfig()
	defer func() { apiClient, accountFlag, autoServe, cfg = oldClient, oldAccount, oldAuto, oldCfg }()

	if err := evaluateLocal("./policies", "my-local-profile"); err != nil {
		t.Fatal(err)
	}
	if sent["profile"] != "my-local-profile" {
		t.Fatalf("the local profile must be sent: %v", sent)
	}
	if _, present := sent["account_id"]; present {
		t.Fatalf("local mode must never send account_id: %v", sent)
	}
}

func TestShortIDNeverPanics(t *testing.T) {
	for in, want := range map[string]string{
		"":                                     "",
		"e1":                                   "e1",
		"12345678":                             "12345678",
		"0a1b2c3d-1111-2222-3333-444444444444": "0a1b2c3d",
	} {
		if got := shortID(in); got != want {
			t.Errorf("shortID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDiffRefusesTwoDifferentAccounts(t *testing.T) {
	msg := differentAccountsMessage(acctDev, "dev", acctPayer, "payer")
	for _, want := range []string{"different AWS accounts", "dev (" + acctDev + ")", "payer (" + acctPayer + ")"} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %q", want, msg)
		}
	}
}

func TestDiffAllowsSameAccountOrUnknown(t *testing.T) {
	for _, c := range [][4]string{
		{acctDev, "dev", acctDev, "dev"},
		{acctDev, "dev", "", ""}, // one predates account tracking: cannot tell, do not block
		{"", "", acctPayer, "payer"},
		{"", "", "", ""},
	} {
		if got := differentAccountsMessage(c[0], c[1], c[2], c[3]); got != "" {
			t.Errorf("%v blocked: %q", c, got)
		}
	}
}

func TestListingKeepsEveryAccountUnlessFiltered(t *testing.T) {
	c, _ := listServer(t, mixedList)
	all, err := fetchEvaluationList(c, 15, "", false)
	if err != nil || ids(all.Evaluations) != "p2,d2,p1" || all.Note != "" {
		t.Fatalf("a plain listing must keep every account: %v %s note=%q", err, ids(all.Evaluations), all.Note)
	}
	one, err := fetchEvaluationList(c, 15, acctDev, false)
	if err != nil || ids(one.Evaluations) != "d2" {
		t.Fatalf("--account must narrow it: %v %s", err, ids(one.Evaluations))
	}
}
