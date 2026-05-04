package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type CrtSh struct{}

func (c *CrtSh) Name() string        { return "crt.sh" }
func (c *CrtSh) ID() string          { return "crtsh" }
func (c *CrtSh) NeedsKey() bool      { return false }
func (c *CrtSh) IsAvailable() bool   { return true }
func (c *CrtSh) DefaultTimeout() int { return 0 }

func (c *CrtSh) Run(ctx context.Context, domain string) ([]string, error) {
	// small random delay 0-500ms before first attempt
	jitter := time.Duration(rand.Intn(500)) * time.Millisecond
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(jitter):
	}

	results, err := c.fetch(ctx, domain)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}
		results, err = c.fetch(ctx, domain)
	}
	return results, err
}

func (c *CrtSh) fetch(ctx context.Context, domain string) ([]string, error) {
	reqURL := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json&excluded=expired", domain)
	if os.Getenv("SCION_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] crtsh URL: %s\n", reqURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 502 || resp.StatusCode == 503 {
		return nil, fmt.Errorf("crtsh: upstream unavailable (HTTP %d) — crt.sh may be experiencing an outage", resp.StatusCode)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("crtsh: unexpected status code %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		return nil, fmt.Errorf("crtsh: unexpected content-type %q", ct)
	}

	var results []struct {
		NameValue  string `json:"name_value"`
		CommonName string `json:"common_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("crtsh: decode error: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	addNames := func(raw string) {
		for _, name := range strings.Split(raw, "\n") {
			name = cleanDomain(name)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	for _, r := range results {
		addNames(r.NameValue)
		addNames(r.CommonName)
	}
	return out, nil
}
