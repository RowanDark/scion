package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type CrtSh struct{}

func (c *CrtSh) Name() string        { return "crt.sh" }
func (c *CrtSh) ID() string          { return "crtsh" }
func (c *CrtSh) NeedsKey() bool      { return false }
func (c *CrtSh) IsAvailable() bool   { return true }

func (c *CrtSh) Run(ctx context.Context, domain string) ([]string, error) {
	url := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", domain)
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
		Name string `json:"name_value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("crtsh: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, r := range results {
		for _, name := range strings.Split(r.Name, "\n") {
			name = cleanDomain(name)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out, nil
}
