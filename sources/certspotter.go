package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type CertSpotter struct{}

func (c *CertSpotter) Name() string        { return "Certspotter" }
func (c *CertSpotter) ID() string          { return "certspotter" }
func (c *CertSpotter) NeedsKey() bool      { return false }
func (c *CertSpotter) IsAvailable() bool   { return true }
func (c *CertSpotter) DefaultTimeout() int { return 0 }

func (c *CertSpotter) Run(ctx context.Context, domain string) ([]string, error) {
	url := fmt.Sprintf("https://api.certspotter.com/v1/issuances?domain=%s&include_subdomains=true&expand=dns_names", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []struct {
		DNSNames []string `json:"dns_names"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("certspotter: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, r := range results {
		for _, name := range r.DNSNames {
			name = cleanDomain(name)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out, nil
}
