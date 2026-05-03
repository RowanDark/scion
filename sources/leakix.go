package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type LeakIX struct{}

func (l *LeakIX) Name() string        { return "LeakIX" }
func (l *LeakIX) ID() string          { return "leakix" }
func (l *LeakIX) NeedsKey() bool      { return true }
func (l *LeakIX) IsAvailable() bool   { return os.Getenv("LEAKIX_API_KEY") != "" }
func (l *LeakIX) DefaultTimeout() int { return 0 }

func (l *LeakIX) Run(ctx context.Context, domain string) ([]string, error) {
	key := os.Getenv("LEAKIX_API_KEY")

	url := fmt.Sprintf("https://leakix.net/api/subdomains/%s", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("api-key", key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 404 means no data for this domain — not an error condition
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("leakix: unexpected status %d", resp.StatusCode)
	}

	var results []struct {
		Subdomain string `json:"Subdomain"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("leakix: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, r := range results {
		name := cleanDomain(r.Subdomain)
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}
