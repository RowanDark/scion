package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// ReconeerSource is an opt-in keyed source: the free tier is capped at 10
// queries/day, so it stays silent and unavailable until RECONEER_API_KEY is set.
type ReconeerSource struct{}

func (r *ReconeerSource) Name() string        { return "Reconeer" }
func (r *ReconeerSource) ID() string          { return "reconeer" }
func (r *ReconeerSource) NeedsKey() bool      { return true }
func (r *ReconeerSource) DefaultTimeout() int { return 0 }
func (r *ReconeerSource) IsAvailable() bool   { return os.Getenv("RECONEER_API_KEY") != "" }

var reconeerBaseURL = "https://www.reconeer.com/api/domain"

func (r *ReconeerSource) Run(ctx context.Context, domain string) ([]string, error) {
	key := os.Getenv("RECONEER_API_KEY")
	if key == "" {
		return nil, errors.New("reconeer: RECONEER_API_KEY not set")
	}

	reqURL := fmt.Sprintf("%s/%s", reconeerBaseURL, domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
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

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("reconeer: rate limit exceeded (free tier: 10 queries/day)")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("reconeer: unauthorized — check RECONEER_API_KEY")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reconeer: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Subdomains []string `json:"subdomains"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("reconeer: decode error: %w", err)
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string
	for _, sub := range result.Subdomains {
		name := cleanDomain(sub)
		if name == "" {
			continue
		}
		if name != domain && !strings.HasSuffix(name, suffix) {
			continue
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}
