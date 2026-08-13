package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ThreatCrowd is largely defunct as an upstream service; it is retained for
// compatibility and degrades to an empty result set rather than failing hard
// when the API no longer responds usefully.
type ThreatCrowd struct{}

func (t *ThreatCrowd) Name() string        { return "ThreatCrowd" }
func (t *ThreatCrowd) ID() string          { return "threatcrowd" }
func (t *ThreatCrowd) NeedsKey() bool      { return false }
func (t *ThreatCrowd) IsAvailable() bool   { return true }
func (t *ThreatCrowd) DefaultTimeout() int { return 30 }

var threatcrowdAPIURL = "http://ci-www.threatcrowd.org/searchApi/v2/domain/report/"

func (t *ThreatCrowd) Run(ctx context.Context, domain string) ([]string, error) {
	reqURL := fmt.Sprintf("%s?domain=%s", threatcrowdAPIURL, domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("threatcrowd: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Subdomains []string `json:"subdomains"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("threatcrowd: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, sub := range result.Subdomains {
		name := cleanDomain(sub)
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}
