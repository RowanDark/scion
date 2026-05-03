package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type AlienVault struct{}

func (a *AlienVault) Name() string      { return "AlienVault OTX" }
func (a *AlienVault) ID() string        { return "alienvault" }
func (a *AlienVault) NeedsKey() bool    { return false }
func (a *AlienVault) IsAvailable() bool { return true }

func (a *AlienVault) Run(ctx context.Context, domain string) ([]string, error) {
	url := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/passive_dns", domain)
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

	var result struct {
		PassiveDNS []struct {
			Hostname string `json:"hostname"`
		} `json:"passive_dns"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("alienvault: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, entry := range result.PassiveDNS {
		name := cleanDomain(entry.Hostname)
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}
