package sources

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
)

type HackerTarget struct{}

func (h *HackerTarget) Name() string      { return "HackerTarget" }
func (h *HackerTarget) ID() string        { return "hackertarget" }
func (h *HackerTarget) NeedsKey() bool    { return false }
func (h *HackerTarget) IsAvailable() bool { return true }

func (h *HackerTarget) Run(ctx context.Context, domain string) ([]string, error) {
	url := fmt.Sprintf("https://api.hackertarget.com/hostsearch/?q=%s", domain)
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

	seen := make(map[string]bool)
	var out []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		// format: subdomain,ip
		parts := strings.SplitN(line, ",", 2)
		if len(parts) < 1 {
			continue
		}
		name := cleanDomain(parts[0])
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, scanner.Err()
}
