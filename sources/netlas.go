package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type NetlasSource struct{}

func (s *NetlasSource) Name() string        { return "Netlas" }
func (s *NetlasSource) ID() string          { return "netlas" }
func (s *NetlasSource) NeedsKey() bool      { return true }
func (s *NetlasSource) DefaultTimeout() int { return 30 }
func (s *NetlasSource) IsAvailable() bool   { return os.Getenv("NETLAS_API_KEY") != "" }

func (s *NetlasSource) Run(ctx context.Context, domain string) ([]string, error) {
	apiKey := os.Getenv("NETLAS_API_KEY")

	type item struct {
		Data struct {
			Domain string `json:"domain"`
		} `json:"data"`
	}
	type response struct {
		Items []item `json:"items"`
	}

	seen := make(map[string]bool)
	var out []string
	suffix := "." + domain

	const pageSize = 100
	const maxPages = 3

	for page := 0; page < maxPages; page++ {
		start := page * pageSize
		query := url.QueryEscape(fmt.Sprintf("domain:*.%s", domain))
		reqURL := fmt.Sprintf(
			"https://app.netlas.io/api/dns/?q=%s&source_type=include&start=%d&fields=domain",
			query, start,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return out, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return out, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return out, err
		}

		if resp.StatusCode == 429 {
			return out, &PartialResultError{Reason: "netlas: daily quota exceeded (50 requests/day on free tier)"}
		}
		if resp.StatusCode == 403 || resp.StatusCode == 401 {
			return out, fmt.Errorf("netlas: unauthorized — check NETLAS_API_KEY")
		}
		if resp.StatusCode != http.StatusOK {
			return out, fmt.Errorf("netlas: unexpected status %d", resp.StatusCode)
		}

		var result response
		if err := json.Unmarshal(body, &result); err != nil {
			return out, fmt.Errorf("netlas: json decode: %w", err)
		}

		if len(result.Items) == 0 {
			break
		}

		for _, it := range result.Items {
			name := cleanDomain(it.Data.Domain)
			if name == "" {
				continue
			}
			if strings.HasSuffix(name, suffix) || name == domain {
				if !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
		}

		if page < maxPages-1 {
			select {
			case <-time.After(200 * time.Millisecond):
			case <-ctx.Done():
				return out, &PartialResultError{Reason: "netlas: context cancelled"}
			}
		}
	}

	return out, nil
}
