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
	// AccountID is the connected AWS account to evaluate (cloud mode). Required by api.sofe.dev when
	// more than one account is connected; omitted for the local server.
	AccountID string `json:"account_id,omitempty"`
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

	// Account and cost data (api.sofe.dev, S111). Empty/nil when the server is older or has no data:
	// an unknown cost is never reported as 0.
	AccountID          string   `json:"account_id,omitempty"`
	AccountAlias       string   `json:"account_alias,omitempty"`
	CostStatus         string   `json:"cost_status,omitempty"`
	CostDetail         string   `json:"cost_detail,omitempty"`
	CostCoverageUSD    *float64 `json:"cost_coverage_usd,omitempty"`
	CostAccountMonthly *float64 `json:"cost_account_monthly,omitempty"`
	CostMatchedMonthly *float64 `json:"cost_matched_monthly,omitempty"`
}

// AccountRef is a connected AWS account as api.sofe.dev lists it in an account_required answer.
type AccountRef struct {
	AccountID string `json:"account_id"`
	Alias     string `json:"alias"`
}

// APIError is an error answer (HTTP >= 400) from the API. Error() keeps the original
// "API error <status>: <body>" format, so callers that print it see no change.
type APIError struct {
	Status   int
	Code     string       // the "error" field of the JSON body, when there is one
	Message  string       // the "message" field, when there is one
	Accounts []AccountRef // set on 400 account_required
	Body     string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Status, e.Body)
}

func newAPIError(status int, body []byte) *APIError {
	e := &APIError{Status: status, Body: string(body)}
	var parsed struct {
		Error    string       `json:"error"`
		Message  string       `json:"message"`
		Accounts []AccountRef `json:"accounts"`
	}
	// A body that is not JSON (an HTML gateway error, for example) simply leaves the fields empty.
	if json.Unmarshal(body, &parsed) == nil {
		e.Code, e.Message, e.Accounts = parsed.Error, parsed.Message, parsed.Accounts
	}
	return e
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
		return nil, newAPIError(resp.StatusCode, data)
	}
	return data, nil
}

// Get performs a GET request to the given path and returns raw bytes
func (c *Client) Get(path string) ([]byte, error) {
	return c.do("GET", path, nil)
}

// Post performs a POST request with the given body and returns raw bytes
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	return c.do("POST", path, body)
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
