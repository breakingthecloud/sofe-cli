package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseProfileMap(t *testing.T) {
	got := parseProfileMap(" 111111111111=prod, 222222222222=dev ,bad,333333333333= ")
	want := map[string]string{"111111111111": "prod", "222222222222": "dev"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseProfileMap = %#v, want %#v", got, want)
	}
}

func TestProfileMapFromCandidatesSkipsUnknownProfiles(t *testing.T) {
	known := map[string]string{"111111111111": "prod", "222222222222": "dev"}
	got := profileMapFromCandidates([]string{"prod", "expired", "dev"}, func(profile string) (string, bool) {
		for account, candidate := range known {
			if candidate == profile {
				return account, true
			}
		}
		return "", false
	})
	want := map[string]string{"111111111111": "prod", "222222222222": "dev"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("profileMapFromCandidates = %#v, want %#v", got, want)
	}
}

func TestUpdateStackArgsKeepTheExternalID(t *testing.T) {
	args := strings.Join(updateStackArgs("prod", "us-east-1", "sofe-read-only", "https://example.invalid/t.yaml"), " ")
	for _, want := range []string{
		"cloudformation update-stack",
		"--stack-name sofe-read-only",
		"--template-url https://example.invalid/t.yaml",
		"--parameters ParameterKey=ExternalId,UsePreviousValue=true",
		"--capabilities CAPABILITY_NAMED_IAM",
		"--region us-east-1",
		"--profile prod",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("updateStackArgs missing %q in %q", want, args)
		}
	}
}

func TestMayNeedStackUpdate(t *testing.T) {
	for connectedAt, want := range map[string]bool{
		"2026-09-02T03:08:31.000Z": true,
		"2026-11-15T23:59:59.000Z": true,
		"2026-11-16T00:00:00.000Z": false,
		"2026-11-20T10:00:00.000Z": false,
		"":                         false,
		"not-a-date":               false,
	} {
		if got := mayNeedStackUpdate(connectedAt); got != want {
			t.Errorf("mayNeedStackUpdate(%q) = %v, want %v", connectedAt, got, want)
		}
	}
}

func TestShortDate(t *testing.T) {
	if got := shortDate("2026-09-02T03:08:31.000Z"); got != "2026-09-02" {
		t.Fatalf("shortDate = %q", got)
	}
	if got := shortDate("garbage"); got != "garbage" {
		t.Fatalf("shortDate fallback = %q", got)
	}
}

func TestShellCommandQuotesArgumentsWithSpaces(t *testing.T) {
	command := shellCommand([]string{"aws", "cloudformation", "update-stack", "--profile", "my profile"})
	if !strings.Contains(command, `"my profile"`) {
		t.Fatalf("shellCommand did not quote the profile: %q", command)
	}
}

func TestFirstLine(t *testing.T) {
	if got := firstLine("Stack is in UPDATE_COMPLETE\nextra"); got != "Stack is in UPDATE_COMPLETE" {
		t.Fatalf("firstLine = %q", got)
	}
	if got := firstLine(""); got != "aws command failed" {
		t.Fatalf("firstLine empty = %q", got)
	}
}
