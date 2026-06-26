package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 120 * time.Second},
	}
}

type EvaluateRequest struct {
	PoliciesDir   string   `json:"policies_dir"`
	Profile       string   `json:"profile,omitempty"`
	ResourceTypes []string `json:"resource_types,omitempty"`
	FailOn        string   `json:"fail_on,omitempty"`
}

type Finding struct {
	PolicyName      string   `json:"policy_name"`
	Severity        string   `json:"severity"`
	ResourceID      string   `json:"resource_id"`
	ResourceType    string   `json:"resource_type"`
	Region          string   `json:"region"`
	Message         string   `json:"message"`
	EstimatedSavings *float64 `json:"estimated_savings"`
	Recommendation  *string  `json:"recommendation"`
}

type EvaluateResponse struct {
	EvaluationID         string    `json:"evaluation_id"`
	Timestamp            string    `json:"timestamp"`
	PoliciesEvaluated    int       `json:"policies_evaluated"`
	ResourcesScanned     int       `json:"resources_scanned"`
	FindingsCount        int       `json:"findings_count"`
	TotalEstimatedSavings float64  `json:"total_estimated_savings"`
	Failed               bool      `json:"failed"`
	Findings             []Finding `json:"findings"`
}

type PolicySummary struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Severity      string   `json:"severity"`
	ResourceTypes []string `json:"resource_types"`
	Metric        string   `json:"metric"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (c *Client) do(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w (is sofe-server running on %s?)", err, c.BaseURL)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *Client) Health() (*HealthResponse, error) {
	data, err := c.do("GET", "/health", nil)
	if err != nil {
		return nil, err
	}
	var r HealthResponse
	json.Unmarshal(data, &r)
	return &r, nil
}

func (c *Client) Evaluate(req EvaluateRequest) (*EvaluateResponse, error) {
	data, err := c.do("POST", "/evaluate", req)
	if err != nil {
		return nil, err
	}
	var r EvaluateResponse
	json.Unmarshal(data, &r)
	return &r, nil
}

func (c *Client) Policies(policiesDir string) ([]PolicySummary, error) {
	data, err := c.do("GET", "/policies?policies_dir="+policiesDir, nil)
	if err != nil {
		return nil, err
	}
	var r []PolicySummary
	json.Unmarshal(data, &r)
	return r, nil
}
