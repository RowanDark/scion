package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Wayback struct{}

func (w *Wayback) Name() string        { return "Wayback Machine" }
func (w *Wayback) ID() string          { return "wayback" }
func (w *Wayback) NeedsKey() bool      { return false }
func (w *Wayback) IsAvailable() bool   { return true }
func (w *Wayback) DefaultTimeout() int { return 90 }

func (w *Wayback) Run(ctx context.Context, domain string) ([]string, error) {
	fromYear := time.Now().Year() - 3
	apiURL := fmt.Sprintf(
		"https://web.archive.org/cdx/search/cdx?url=*.%s&output=json&fl=original&collapse=urlkey&limit=10000&from=%d0101",
		domain,
		fromYear,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("wayback: failed to read response: %v", err)
	}

	bodyStr := strings.TrimSpace(string(body))
	if strings.HasPrefix(bodyStr, "<") {
		return nil, fmt.Errorf(
			"wayback: received HTML instead of JSON — CDX API may be overloaded or throttling this IP. " +
				"Try again later or use --timeout to extend the deadline.",
		)
	}

	var rows [][]string
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("wayback: failed to parse response: %v", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		raw := row[0]
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
