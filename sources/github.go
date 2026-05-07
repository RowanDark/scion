package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"
)

type GitHubSource struct{}

func (s *GitHubSource) Name() string        { return "GitHub" }
func (s *GitHubSource) ID() string          { return "github" }
func (s *GitHubSource) NeedsKey() bool      { return true }
func (s *GitHubSource) DefaultTimeout() int { return 60 }
func (s *GitHubSource) IsAvailable() bool {
	return os.Getenv("GITHUB_TOKEN") != ""
}

func (s *GitHubSource) Run(ctx context.Context, domain string) ([]string, error) {
	token := os.Getenv("GITHUB_TOKEN")

	pattern := regexp.MustCompile(
		`(?i)[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?\.` + regexp.QuoteMeta(domain),
	)

	seen := make(map[string]bool)
	var out []string

	query := url.QueryEscape(domain + " in:file")

	for page := 1; page <= 10; page++ {
		apiURL := fmt.Sprintf(
			"https://api.github.com/search/code?q=%s&per_page=100&page=%d",
			query, page,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return out, err
		}
		req.Header.Set("Authorization", "token "+token)
		req.Header.Set("Accept", "application/vnd.github.text-match+json")
		req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return out, err
		}

		remaining := resp.Header.Get("X-RateLimit-Remaining")
		if remaining == "0" {
			resp.Body.Close()
			return out, fmt.Errorf("github: rate limit exhausted — wait before retrying")
		}

		if resp.StatusCode == 403 {
			retryAfter := resp.Header.Get("Retry-After")
			resp.Body.Close()
			if retryAfter != "" {
				return out, fmt.Errorf("github: secondary rate limit hit — retry after %s seconds", retryAfter)
			}
			return out, fmt.Errorf("github: forbidden (403) — query may be too broad or token lacks required scope")
		}
		if resp.StatusCode == 401 {
			resp.Body.Close()
			return out, fmt.Errorf("github: unauthorized (401) — check GITHUB_TOKEN is set correctly")
		}
		if resp.StatusCode == 422 {
			resp.Body.Close()
			break
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return out, fmt.Errorf("github: unexpected status %d", resp.StatusCode)
		}

		var result struct {
			Items []struct {
				TextMatches []struct {
					Fragment string `json:"fragment"`
				} `json:"text_matches"`
			} `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return out, fmt.Errorf("github: decode error: %w", err)
		}
		resp.Body.Close()

		if len(result.Items) == 0 {
			break
		}

		for _, item := range result.Items {
			for _, match := range item.TextMatches {
				for _, found := range pattern.FindAllString(match.Fragment, -1) {
					name := cleanDomain(found)
					if name != "" && !seen[name] {
						seen[name] = true
						out = append(out, name)
					}
				}
			}
		}

		time.Sleep(2 * time.Second)
	}

	return out, nil
}
