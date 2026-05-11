package sources

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type RobtexSource struct{}

func (s *RobtexSource) Name() string        { return "Robtex" }
func (s *RobtexSource) ID() string          { return "robtex" }
func (s *RobtexSource) NeedsKey() bool      { return false }
func (s *RobtexSource) DefaultTimeout() int { return 30 }
func (s *RobtexSource) IsAvailable() bool   { return true }

func (s *RobtexSource) Run(ctx context.Context, domain string) ([]string, error) {
	// Robtex requests a gentle rate — small delay before querying.
	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	reqURL := fmt.Sprintf("https://freeapi.robtex.com/pdns/forward/%s", domain)
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
		return nil, fmt.Errorf("robtex: unexpected status %d", resp.StatusCode)
	}

	// Response is newline-delimited JSON objects, not a JSON array.
	type record struct {
		RRName string `json:"rrname"`
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec record
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		name := cleanDomain(rec.RRName)
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
	return out, scanner.Err()
}
