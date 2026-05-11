package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type DigitorusSource struct{}

func (s *DigitorusSource) Name() string        { return "Digitorus" }
func (s *DigitorusSource) ID() string          { return "digitorus" }
func (s *DigitorusSource) NeedsKey() bool      { return false }
func (s *DigitorusSource) DefaultTimeout() int { return 45 }
func (s *DigitorusSource) IsAvailable() bool   { return true }

func (s *DigitorusSource) Run(ctx context.Context, domain string) ([]string, error) {
	// Try the JSON endpoint first; fall back to HTML parsing if it fails.
	results, err := s.tryJSON(ctx, domain)
	if err == nil && len(results) > 0 {
		return results, nil
	}
	return s.tryHTML(ctx, domain)
}

// tryJSON attempts the digitorus.com JSON API endpoint.
func (s *DigitorusSource) tryJSON(ctx context.Context, domain string) ([]string, error) {
	reqURL := fmt.Sprintf("https://www.digitorus.com/api/ct?domain=%s", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("digitorus json: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Subdomains []string `json:"subdomains"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("digitorus json: decode: %w", err)
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string
	for _, entry := range result.Subdomains {
		name := cleanDomain(entry)
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
	return out, nil
}

// tryHTML falls back to certificatedetails.com HTML parsing.
func (s *DigitorusSource) tryHTML(ctx context.Context, domain string) ([]string, error) {
	reqURL := fmt.Sprintf("https://certificatedetails.com/%s", domain)
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
		return nil, fmt.Errorf("digitorus html: status %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("digitorus html: parse: %w", err)
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if strings.HasSuffix(text, suffix) || text == domain {
				name := cleanDomain(text)
				if name != "" && !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out, nil
}
