package sources

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"regexp"
)

// Digitorus queries certificatedetails.com (digitorus' certificate transparency
// lookup service) and regex-extracts hostnames from the returned page text.
type Digitorus struct{}

func (d *Digitorus) Name() string        { return "Digitorus" }
func (d *Digitorus) ID() string          { return "digitorus" }
func (d *Digitorus) NeedsKey() bool      { return false }
func (d *Digitorus) IsAvailable() bool   { return true }
func (d *Digitorus) DefaultTimeout() int { return 0 }

var digitorusBaseURL = "https://certificatedetails.com"

func (d *Digitorus) Run(ctx context.Context, domain string) ([]string, error) {
	reqURL := fmt.Sprintf("%s/%s", digitorusBaseURL, domain)
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

	// certificatedetails.com returns 404 for domains with no direct cert
	// history, but the 404 page body itself still lists related subdomains
	// pulled from adjacent certificates — so 404 is not treated as an error.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("digitorus: unexpected status %d", resp.StatusCode)
	}

	pattern := regexp.MustCompile(`(?i)[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?\.` + regexp.QuoteMeta(domain))

	seen := make(map[string]bool)
	var out []string
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return out, &PartialResultError{Reason: "digitorus: context cancelled"}
		}
		for _, found := range pattern.FindAllString(scanner.Text(), -1) {
			name := cleanDomain(found)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if len(out) > 0 {
			return out, &PartialResultError{Reason: fmt.Sprintf("digitorus: %v", err)}
		}
		return nil, fmt.Errorf("digitorus: %w", err)
	}

	return out, nil
}
