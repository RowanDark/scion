package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type BeVigilSource struct{}

func (s *BeVigilSource) Name() string        { return "BeVigil" }
func (s *BeVigilSource) ID() string          { return "bevigil" }
func (s *BeVigilSource) NeedsKey() bool      { return true }
func (s *BeVigilSource) DefaultTimeout() int { return 30 }
func (s *BeVigilSource) IsAvailable() bool   { return os.Getenv("BEVIGIL_API_KEY") != "" }

func (s *BeVigilSource) Run(ctx context.Context, domain string) ([]string, error) {
	apiKey := os.Getenv("BEVIGIL_API_KEY")

	reqURL := fmt.Sprintf("https://osint.bevigil.com/api/%s/subdomains/", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Access-Token", apiKey)
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("bevigil: unauthorized — check BEVIGIL_API_KEY")
	}
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("bevigil: rate limit exceeded")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bevigil: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Domain     string   `json:"domain"`
		Subdomains []string `json:"subdomains"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("bevigil: json decode: %w", err)
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
