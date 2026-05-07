package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type URLScanSource struct{}

func (s *URLScanSource) Name() string        { return "URLScan.io" }
func (s *URLScanSource) ID() string          { return "urlscan" }
func (s *URLScanSource) NeedsKey() bool      { return true }
func (s *URLScanSource) DefaultTimeout() int { return 30 }
func (s *URLScanSource) IsAvailable() bool {
	return os.Getenv("URLSCAN_API_KEY") != ""
}

func (s *URLScanSource) Run(ctx context.Context, domain string) ([]string, error) {
	reqURL := fmt.Sprintf(
		"https://urlscan.io/api/v1/search/?q=domain:%s&size=100&fields=page.domain",
		domain,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("urlscan: failed to create request: %v", err)
	}

	req.Header.Set("API-Key", os.Getenv("URLSCAN_API_KEY"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("urlscan: request failed: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 429:
		return nil, fmt.Errorf("urlscan: rate limit exceeded — free tier allows 1,000 searches/day")
	case 401:
		return nil, fmt.Errorf("urlscan: unauthorized — check URLSCAN_API_KEY")
	case 400:
		return nil, fmt.Errorf("urlscan: bad request — check query format")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("urlscan: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("urlscan: failed to read response: %v", err)
	}

	var result urlscanResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("urlscan: failed to parse response: %v", err)
	}

	seen := make(map[string]struct{})
	var domains []string

	for _, r := range result.Results {
		d := strings.ToLower(strings.TrimSpace(r.Page.Domain))
		if d == "" {
			continue
		}
		if d != domain && !strings.HasSuffix(d, "."+domain) {
			continue
		}
		if _, exists := seen[d]; !exists {
			seen[d] = struct{}{}
			domains = append(domains, d)
		}
	}

	return domains, nil
}

type urlscanResponse struct {
	Results []struct {
		Page struct {
			Domain string `json:"domain"`
		} `json:"page"`
	} `json:"results"`
	Total   int  `json:"total"`
	HasMore bool `json:"has_more"`
}
