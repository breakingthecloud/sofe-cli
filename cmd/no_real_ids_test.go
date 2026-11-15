package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// This repository is public. Tests and docs must never carry the AWS account IDs (or real figures) of the people who
// work on it. Any 12-digit number in the repo must be an obviously invented one: the same digit repeated
// (111111111111, 222222222222, ...) or AWS' documentation example 123456789012.
// The test does not (and must not) list real IDs: it only accepts invented ones.

var twelveDigits = regexp.MustCompile(`(^|[^0-9])([0-9]{12})($|[^0-9])`)

func isInventedAccountID(id string) bool {
	if id == "123456789012" {
		return true
	}
	return strings.Count(id, string(id[0])) == len(id) // 111111111111, 000000000000, ...
}

func TestTheInventedIDRule(t *testing.T) {
	for id, want := range map[string]bool{
		"111111111111": true, "222222222222": true, "000000000000": true, "123456789012": true,
		"987654321098": false, "135792468013": false, "123456789013": false, "111111111112": false,
	} {
		if got := isInventedAccountID(id); got != want {
			t.Errorf("isInventedAccountID(%s) = %v, want %v", id, got, want)
		}
	}
}

func TestNoRealLookingAccountIDsInTheRepository(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var offenders []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".md", ".yaml", ".yml", ".sh", ".json", ".txt", ".tf":
		default:
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(root, path)
		for i, line := range strings.Split(string(data), "\n") {
			for _, m := range twelveDigits.FindAllStringSubmatch(line, -1) {
				if !isInventedAccountID(m[2]) && rel != "cmd/no_real_ids_test.go" {
					// Report where, never the value: the point is to keep it out of logs too.
					offenders = append(offenders, rel+":"+itoa(i+1))
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Fatalf("12-digit numbers that are not obviously invented (use 111111111111, 222222222222 or 123456789012):\n  %s", strings.Join(offenders, "\n  "))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
