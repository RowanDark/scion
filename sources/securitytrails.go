package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

type SecurityTrails struct{}

func (s *SecurityTrails) Name() string        { return "SecurityTrails" }
func (s *SecurityTrails) ID() string          { return "securitytrails" }
func (s *SecurityTrails) NeedsKey() bool      { return true }
func (s *SecurityTrails) IsAvailable() bool   { return os.Getenv("ST_API_KEY") != "" }
func (s *SecurityTrails) DefaultTimeout() int { return 0 }

func (s *SecurityTrails) Run(ctx context.Context, domain string) ([]string, error) {
	key := os.Getenv("ST_API_KEY")
	if key == "" {
		return nil, errors.New("securitytrails: ST_API_KEY not set")
	}

	url := fmt.Sprintf("https://api.securitytrails.com/v1/domain/%s/subdomains", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("APIKEY", key)
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Subdomains []string `json:"subdomains"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("securitytrails: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	for _, sub := range result.Subdomains {
		name := cleanDomain(sub + "." + domain)
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}
