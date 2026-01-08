package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	baseURL    = "https://context7.com"
	searchPath = "/api/v2/libs/search"
)

// Library represents a library result from context7.com
type Library struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Branch          string   `json:"branch"`
	LastUpdateDate  string   `json:"lastUpdateDate"`
	State           string   `json:"state"`
	TotalTokens     int      `json:"totalTokens"`
	TotalSnippets   int      `json:"totalSnippets"`
	Stars           int      `json:"stars"`
	TrustScore      float64  `json:"trustScore"`
	BenchmarkScore  float64  `json:"benchmarkScore"`
	Versions        []string `json:"versions"`
	Score           float64  `json:"score"`
	VIP             bool     `json:"vip"`
}

// SearchResponse represents the API response from the search endpoint
type SearchResponse struct {
	Results []Library `json:"results"`
}

// Client is an HTTP client for context7.com
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new context7 API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchLibraries searches for libraries matching the query
func (c *Client) SearchLibraries(query string) ([]Library, error) {
	// Build search URL
	searchURL := fmt.Sprintf("%s%s?query=%s", baseURL, searchPath, url.QueryEscape(query))

	// Make HTTP request
	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make search request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return searchResp.Results, nil
}

// FetchLLMsTxt fetches the llms.txt content for a library
func (c *Client) FetchLLMsTxt(libraryID string) (string, error) {
	// Build llms.txt URL
	llmsURL := fmt.Sprintf("%s%s/llms.txt", baseURL, libraryID)

	// Make HTTP request
	resp, err := c.httpClient.Get(llmsURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch llms.txt: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llms.txt request failed with status %d", resp.StatusCode)
	}

	// Read content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read llms.txt content: %w", err)
	}

	return string(content), nil
}
