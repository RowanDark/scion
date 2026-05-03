package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type CrtSh struct{}

func (c *CrtSh) Name() string        { return "crt.sh" }
func (c *CrtSh) ID() string          { return "crtsh" }
func (c *CrtSh) NeedsKey() bool      { return false }
func (c *CrtSh) IsAvailable() bool   { return true }
func (c *CrtSh) DefaultTimeout() int { return 0 }

func (c *CrtSh) Run(ctx context.Context, domain string) ([]string, error) {
	results, err := c.fetch(ctx, domain)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(3 * time.Second):
		}
		results, err = c.fetch(ctx, domain)
	}
	return results, err
}

func (c *CrtSh) fetch(ctx context.Context, domain string) ([]string, error) {
	url := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("crtsh: unexpected content-type %q: %s", ct, strings.TrimSpace(string(preview)))
	}

	var results []struct {
		NameValue  string `json:"name_value"`
		CommonName string `json:"common_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("crtsh: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	addNames := func(raw string) {
		for _, name := range strings.Split(raw, "\n") {
			name = cleanDomain(name)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	for _, r := range results {
		addNames(r.NameValue)
		addNames(r.CommonName)
	}
	return out, nil
}
