package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type FullHuntSource struct{}

func (s *FullHuntSource) Name() string        { return "FullHunt" }
func (s *FullHuntSource) ID() string          { return "fullhunt" }
func (s *FullHuntSource) NeedsKey() bool      { return true }
func (s *FullHuntSource) DefaultTimeout() int { return 0 }
func (s *FullHuntSource) IsAvailable() bool {
	return os.Getenv("FULLHUNT_KEY") != ""
}

func (s *FullHuntSource) Run(ctx context.Context, domain string) ([]string, error) {
	key := os.Getenv("FULLHUNT_KEY")

	url := fmt.Sprintf("https://fullhunt.io/api/v1/domain/%s/subdomains", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-KEY", key)
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("fullhunt: invalid API key (401)")
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("fullhunt: rate limit exceeded (429)")
	case http.StatusOK:
		// continue
	default:
		return nil, fmt.Errorf("fullhunt: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Hosts []string `json:"hosts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("fullhunt: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, name := range result.Hosts {
		name = cleanDomain(name)
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}
