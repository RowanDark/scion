package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type HudsonRockSource struct{}

func (h *HudsonRockSource) Name() string        { return "HudsonRock" }
func (h *HudsonRockSource) ID() string          { return "hudsonrock" }
func (h *HudsonRockSource) NeedsKey() bool      { return false }
func (h *HudsonRockSource) IsAvailable() bool   { return true }
func (h *HudsonRockSource) DefaultTimeout() int { return 0 }

var hudsonrockAPIURL = "https://cavalier.hudsonrock.com/api/json/v2/osint-tools-by-domain"

func (h *HudsonRockSource) Run(ctx context.Context, domain string) ([]string, error) {
	reqURL := fmt.Sprintf("%s?domain=%s", hudsonrockAPIURL, domain)
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
		return nil, fmt.Errorf("hudsonrock: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			EmployeesURLs []struct {
				URL string `json:"url"`
			} `json:"employees_urls"`
			ClientsURLs []struct {
				URL string `json:"url"`
			} `json:"clients_urls"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("hudsonrock: decode error: %w", err)
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string
	addURL := func(raw string) {
		u, err := url.Parse(raw)
		if err != nil {
			return
		}
		host := cleanDomain(u.Hostname())
		if host == "" {
			return
		}
		if host != domain && !strings.HasSuffix(host, suffix) {
			return
		}
		if !seen[host] {
			seen[host] = true
			out = append(out, host)
		}
	}
	for _, r := range result.Data.EmployeesURLs {
		addURL(r.URL)
	}
	for _, r := range result.Data.ClientsURLs {
		addURL(r.URL)
	}
	return out, nil
}
