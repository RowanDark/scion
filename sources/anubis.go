package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type AnubisSource struct{}

func (s *AnubisSource) Name() string        { return "Anubis" }
func (s *AnubisSource) ID() string          { return "anubis" }
func (s *AnubisSource) NeedsKey() bool      { return false }
func (s *AnubisSource) DefaultTimeout() int { return 30 }
func (s *AnubisSource) IsAvailable() bool   { return true }

func (s *AnubisSource) Run(ctx context.Context, domain string) ([]string, error) {
	reqURL := fmt.Sprintf("https://jldc.me/anubis/subdomains/%s", domain)
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
		return nil, fmt.Errorf("anubis: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var entries []string
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("anubis: json decode: %w", err)
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string
	for _, entry := range entries {
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
