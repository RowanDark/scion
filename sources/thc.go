package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type THCSource struct{}

func (t *THCSource) Name() string        { return "THC" }
func (t *THCSource) ID() string          { return "thc" }
func (t *THCSource) NeedsKey() bool      { return false }
func (t *THCSource) IsAvailable() bool   { return true }
func (t *THCSource) DefaultTimeout() int { return 60 }

var thcAPIURL = "https://ip.thc.org/api/v1/lookup/subdomains"

// thcMaxPages bounds pagination so a misbehaving next-page token can't loop forever.
const thcMaxPages = 20

func (t *THCSource) Run(ctx context.Context, domain string) ([]string, error) {
	seen := make(map[string]bool)
	var out []string
	pageState := ""

	for page := 0; page < thcMaxPages; page++ {
		if page > 0 {
			select {
			case <-time.After(200 * time.Millisecond):
			case <-ctx.Done():
				return out, &PartialResultError{Reason: "thc: context cancelled"}
			}
		}

		domains, next, err := t.fetchPage(ctx, domain, pageState)
		if err != nil {
			if len(out) > 0 {
				return out, &PartialResultError{Reason: fmt.Sprintf("thc: stopped at page %d: %v", page, err)}
			}
			return nil, err
		}

		for _, name := range domains {
			name = cleanDomain(name)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}

		if next == "" {
			break
		}
		pageState = next
	}

	return out, nil
}

func (t *THCSource) fetchPage(ctx context.Context, domain, pageState string) ([]string, string, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"domain":     domain,
		"page_state": pageState,
		"limit":      1000,
	})
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, thcAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Domains []struct {
			Domain string `json:"domain"`
		} `json:"domains"`
		NextPageState string `json:"next_page_state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("decode error: %w", err)
	}

	names := make([]string, 0, len(result.Domains))
	for _, d := range result.Domains {
		names = append(names, d.Domain)
	}
	return names, result.NextPageState, nil
}
