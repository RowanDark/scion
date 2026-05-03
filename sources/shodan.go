package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

type Shodan struct{}

func (s *Shodan) Name() string      { return "Shodan" }
func (s *Shodan) ID() string        { return "shodan" }
func (s *Shodan) NeedsKey() bool    { return true }
func (s *Shodan) IsAvailable() bool { return os.Getenv("SHODAN_API_KEY") != "" }

func (s *Shodan) Run(ctx context.Context, domain string) ([]string, error) {
	key := os.Getenv("SHODAN_API_KEY")
	if key == "" {
		return nil, errors.New("shodan: SHODAN_API_KEY not set")
	}

	url := fmt.Sprintf("https://api.shodan.io/dns/domain/%s?key=%s", domain, key)
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
		Subdomains []string `json:"subdomains"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("shodan: decode error: %w", err)
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
