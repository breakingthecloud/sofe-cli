package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type updateCache struct {
	CheckedAt string `json:"checked_at"`
	Latest    string `json:"latest"`
}

// checkForUpdate checks GitHub for newer version (1x per 24h, non-blocking)
// Returns update message or empty string
func checkForUpdate() string {
	if Version == "dev" || Version == "" {
		return ""
	}

	cache := loadUpdateCache()
	if cache != nil {
		// Check if we already checked within 24h
		checkedAt, err := time.Parse(time.RFC3339, cache.CheckedAt)
		if err == nil && time.Since(checkedAt) < 24*time.Hour {
			// Use cached result
			if cache.Latest != "" && cache.Latest != "v"+Version && isNewer(cache.Latest, "v"+Version) {
				return formatUpdateMessage(cache.Latest)
			}
			return ""
		}
	}

	// Fetch latest from GitHub (with short timeout)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/breakingthecloud/sofe-cli/releases/latest")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	data, _ := io.ReadAll(resp.Body)
	var release struct {
		TagName string `json:"tag_name"`
	}
	json.Unmarshal(data, &release)

	// Save cache
	saveUpdateCache(&updateCache{
		CheckedAt: time.Now().Format(time.RFC3339),
		Latest:    release.TagName,
	})

	if release.TagName != "" && release.TagName != "v"+Version && isNewer(release.TagName, "v"+Version) {
		return formatUpdateMessage(release.TagName)
	}

	return ""
}

func formatUpdateMessage(latest string) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("11")).
		PaddingLeft(1)

	msg := fmt.Sprintf("Update available: v%s → %s\nRun: curl -fsSL https://sofe.dev/install.sh | bash", Version, latest)
	return style.Render(msg)
}

// isNewer returns true if a > b (semver comparison, simplified)
func isNewer(a, b string) bool {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		var av, bv int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &av)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bv)
		}
		if av > bv {
			return true
		}
		if av < bv {
			return false
		}
	}
	return false
}

func getCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sofe", ".update-check")
}

func loadUpdateCache() *updateCache {
	data, err := os.ReadFile(getCachePath())
	if err != nil {
		return nil
	}
	var cache updateCache
	json.Unmarshal(data, &cache)
	return &cache
}

func saveUpdateCache(cache *updateCache) {
	home, _ := os.UserHomeDir()
	os.MkdirAll(filepath.Join(home, ".sofe"), 0700)
	data, _ := json.Marshal(cache)
	os.WriteFile(getCachePath(), data, 0600)
}
