package cmd

import "github.com/breakingthecloud/sofe-cli/internal/client"

// getCloudClient returns an API client configured for cloud mode (api.sofe.dev)
func getCloudClient() *client.Client {
	url := cfg.CloudURL
	if url == "" {
		url = "https://api.sofe.dev"
	}
	return client.New(url, cfg.APIKey)
}

// getAIClient returns an API client for the AI worker (ai-api.sofe.dev)
func getAIClient() *client.Client {
	return client.New("https://ai-api.sofe.dev", cfg.APIKey)
}

// shortID is the first 8 characters of an evaluation ID for display. It never panics on a short or empty ID
// (a server that answers without evaluation_id, or an ID typed by hand).
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
