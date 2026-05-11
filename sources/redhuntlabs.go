package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type RedHuntLabsSource struct{}

func (s *RedHuntLabsSource) Name() string        { return "RedHuntLabs" }
func (s *RedHuntLabsSource) ID() string          { return "redhuntlabs" }
func (s *RedHuntLabsSource) NeedsKey() bool      { return true }
func (s *RedHuntLabsSource) DefaultTimeout() int { return 30 }
func (s *RedHuntLabsSource) IsAvailable() bool   { return os.Getenv("REDHUNTLABS_API_KEY") != "" }

func (s *RedHuntLabsSource) Run(ctx context.Context, domain string) ([]string, error) {
	apiKey := os.Getenv("REDHUNTLABS_API_KEY")

	type response struct {
		Subdomains []string `json:"subdomains"`
		TotalCount int      `json:"total_count"`
	}

	seen := make(map[string]bool)
	var out []string
	suffix := "." + domain

	const pageSize = 100
	const maxResults = 500

	for page := 0; len(out) < maxResults; page++ {
		reqURL := fmt.Sprintf(
			"https://reconapi.redhuntlabs.com/community/v1/domains/subdomains?domain=%s&page=%d&page_size=%d",
			domain, page, pageSize,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return out, err
		}
		req.Header.Set("X-BLOBR-KEY", apiKey)
		req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return out, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return out, err
		}

		if resp.StatusCode == 429 {
			return out, &PartialResultError{Reason: "redhuntlabs: rate limit exceeded"}
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return out, fmt.Errorf("redhuntlabs: unauthorized — check REDHUNTLABS_API_KEY")
		}
		if resp.StatusCode != http.StatusOK {
			return out, fmt.Errorf("redhuntlabs: unexpected status %d", resp.StatusCode)
		}

		var result response
		if err := json.Unmarshal(body, &result); err != nil {
			return out, fmt.Errorf("redhuntlabs: json decode: %w", err)
		}

		if len(result.Subdomains) == 0 {
			break
		}

		for _, entry := range result.Subdomains {
			name := cleanDomain(entry)
			if name == "" {
				continue
			}
			if strings.HasSuffix(name, suffix) || name == domain {
				if !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
		}

		if len(out) >= maxResults {
			break
		}

		select {
		case <-time.After(200 * time.Millisecond):
		case <-ctx.Done():
			return out, &PartialResultError{Reason: "redhuntlabs: context cancelled"}
		}
	}

	return out, nil
}
