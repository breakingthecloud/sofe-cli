package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/breakingthecloud/sofe-cli/internal/client"
	"github.com/spf13/cobra"
)

// Account scope for the commands that READ stored evaluations (history, watch, top, interactive) — S114 Phase D.
//
// These commands never call POST /evaluate, so they never need --account to work. The problem they have is
// different: when several AWS accounts are connected, the evaluation list mixes them, so "the latest evaluation"
// or a 7-point trend belongs to no account in particular. Rules:
//
//   - --account <id> keeps only that account's evaluations.
//   - Without it, if the list holds a single account, nothing changes.
//   - Without it, if the list holds several accounts, the account of the MOST RECENT evaluation is used and the
//     command says so (and lists the others), instead of silently mixing them.
//   - Evaluations stored before account tracking have no account: they are kept while there is one account and
//     left out (and counted) when the list holds several.

// scopeFetchLimit is how many evaluations are fetched before scoping (the API has no per-account filter).
const scopeFetchLimit = 50

type scopedEvaluations struct {
	Evaluations []Evaluation
	// Note explains an automatic choice; "" when nothing was decided for the user.
	Note string
}

func addAccountFilterFlag(c *cobra.Command) {
	c.Flags().String("account", "", "Only evaluations of this AWS account (12 digits). With several accounts, the most recent evaluation's account is used by default")
}

func accountFilter(c *cobra.Command) string {
	v, _ := c.Flags().GetString("account")
	return strings.TrimSpace(v)
}

// accountDisplay is "alias (id)", or just the id, or "" when the evaluation has no account.
func accountDisplay(id, alias string) string {
	switch {
	case id == "":
		return ""
	case alias != "":
		return fmt.Sprintf("%s (%s)", alias, id)
	default:
		return id
	}
}

func scopeEvaluations(evals []Evaluation, requested string) (scopedEvaluations, error) {
	if requested != "" {
		if !validAccountID(requested) {
			return scopedEvaluations{}, fmt.Errorf("--account must be a 12-digit AWS account ID (got %q)", requested)
		}
		var out []Evaluation
		for _, e := range evals {
			if e.AccountID == requested {
				out = append(out, e)
			}
		}
		if len(out) == 0 {
			return scopedEvaluations{}, fmt.Errorf("no evaluations of account %s among your latest %d", requested, len(evals))
		}
		return scopedEvaluations{Evaluations: out}, nil
	}

	var order []string // accounts in order of first appearance (newest first)
	labels := map[string]string{}
	for _, e := range evals {
		if e.AccountID == "" {
			continue
		}
		if _, seen := labels[e.AccountID]; !seen {
			order = append(order, e.AccountID)
			labels[e.AccountID] = accountDisplay(e.AccountID, e.AccountAlias)
		}
	}
	if len(order) <= 1 {
		return scopedEvaluations{Evaluations: evals}, nil
	}

	chosen := order[0]
	var out []Evaluation
	legacy := 0
	for _, e := range evals {
		switch e.AccountID {
		case chosen:
			out = append(out, e)
		case "":
			legacy++
		}
	}
	others := make([]string, 0, len(order)-1)
	for _, id := range order[1:] {
		others = append(others, labels[id])
	}
	note := fmt.Sprintf("Several accounts in your history. Showing %s (the most recent). Others: %s. Choose one with --account <id>.",
		labels[chosen], strings.Join(others, ", "))
	if legacy > 0 {
		note += fmt.Sprintf(" %d older evaluation(s) without an account were left out.", legacy)
	}
	return scopedEvaluations{Evaluations: out, Note: note}, nil
}

// fetchScopedEvaluations lists the user's evaluations, applies the account scope (choosing the most recent
// evaluation's account when several are present) and keeps at most `want`.
func fetchScopedEvaluations(c *client.Client, want int, requested string) (scopedEvaluations, error) {
	return fetchEvaluationList(c, want, requested, true)
}

// fetchEvaluationList is fetchScopedEvaluations with the automatic choice optional: with autoPick=false and no
// --account every account is kept (what a plain listing like `sofe history` wants).
func fetchEvaluationList(c *client.Client, want int, requested string, autoPick bool) (scopedEvaluations, error) {
	if requested != "" && !validAccountID(requested) {
		return scopedEvaluations{}, fmt.Errorf("--account must be a 12-digit AWS account ID (got %q)", requested)
	}
	data, err := c.Get(fmt.Sprintf("/evaluations?limit=%d", scopeFetchLimit))
	if err != nil {
		return scopedEvaluations{}, err
	}
	var hist HistoryResponse
	if err := json.Unmarshal(data, &hist); err != nil {
		return scopedEvaluations{}, fmt.Errorf("unexpected answer from the server: %w", err)
	}
	if len(hist.Evaluations) == 0 {
		return scopedEvaluations{}, nil
	}
	scoped := scopedEvaluations{Evaluations: hist.Evaluations}
	if requested != "" || autoPick {
		var err error
		if scoped, err = scopeEvaluations(hist.Evaluations, requested); err != nil {
			return scopedEvaluations{}, err
		}
	}
	if want > 0 && len(scoped.Evaluations) > want {
		scoped.Evaluations = scoped.Evaluations[:want]
	}
	return scoped, nil
}

// differentAccountsMessage explains why two evaluations cannot be diffed; "" when they can (same account, or
// either one has no account recorded because it predates account tracking).
func differentAccountsMessage(idA, aliasA, idB, aliasB string) string {
	if idA == "" || idB == "" || idA == idB {
		return ""
	}
	return fmt.Sprintf("These evaluations are from different AWS accounts (%s and %s): new/fixed findings between them would mislead. Compare two evaluations of the same account.",
		accountDisplay(idA, aliasA), accountDisplay(idB, aliasB))
}
