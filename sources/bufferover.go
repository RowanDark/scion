package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type BufferOverSource struct{}

func (s *BufferOverSource) Name() string        { return "BufferOver" }
func (s *BufferOverSource) ID() string          { return "bufferover" }
func (s *BufferOverSource) NeedsKey() bool      { return true }
func (s *BufferOverSource) DefaultTimeout() int { return 0 }
func (s *BufferOverSource) IsAvailable() bool {
	return os.Getenv("BUFFEROVER_KEY") != ""
}

func (s *BufferOverSource) Run(ctx context.Context, domain string) ([]string, error) {
	key := os.Getenv("BUFFEROVER_KEY")

	url := fmt.Sprintf("https://tls.bufferover.run/dns?q=.%s", domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bufferover: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Results []string `json:"Results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("bufferover: decode error: %w", err)
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string

	for _, line := range result.Results {
		parts := strings.Split(line, ",")
		for i := 2; i < len(parts); i++ {
			name := cleanDomain(parts[i])
			if name != "" && strings.HasSuffix(name, suffix) && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}

	return out, nil
}
