package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Wayback struct{}

func (w *Wayback) Name() string        { return "Wayback Machine" }
func (w *Wayback) ID() string          { return "wayback" }
func (w *Wayback) NeedsKey() bool      { return false }
func (w *Wayback) IsAvailable() bool   { return true }
func (w *Wayback) DefaultTimeout() int { return 90 }

func (w *Wayback) Run(ctx context.Context, domain string) ([]string, error) {
	apiURL := fmt.Sprintf(
		"https://web.archive.org/cdx/search/cdx?url=*.%s&output=json&fl=original&collapse=urlkey&limit=10000",
		domain,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rows [][]string
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("wayback: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		raw := row[0]
		// skip the header row
		if raw == "original" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Hostname())
		host = strings.TrimSuffix(host, ".")
		if host != "" && !seen[host] {
			seen[host] = true
			out = append(out, host)
		}
	}
	return out, nil
}
