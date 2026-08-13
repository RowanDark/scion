package sources

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type CommonCrawl struct{}

func (c *CommonCrawl) Name() string        { return "CommonCrawl" }
func (c *CommonCrawl) ID() string          { return "commoncrawl" }
func (c *CommonCrawl) NeedsKey() bool      { return false }
func (c *CommonCrawl) IsAvailable() bool   { return true }
func (c *CommonCrawl) DefaultTimeout() int { return 90 }

var commoncrawlCollinfoURL = "https://index.commoncrawl.org/collinfo.json"

// commoncrawlMaxYears caps how many yearly indexes get queried — CommonCrawl
// publishes multiple indexes per year and querying all of them is far too
// slow for a passive source; this mirrors subfinder's cap of one index per
// of the most recent 5 years.
const commoncrawlMaxYears = 5

// commoncrawlPerIndexLimit bounds each index query's result set — CDX indexes
// can return enormous match counts for popular domains.
const commoncrawlPerIndexLimit = 5000

type commoncrawlIndex struct {
	ID     string `json:"id"`
	APIURL string `json:"cdx-api"`
}

func (c *CommonCrawl) Run(ctx context.Context, domain string) ([]string, error) {
	indexes, err := c.recentIndexes(ctx)
	if err != nil {
		return nil, fmt.Errorf("commoncrawl: %w", err)
	}
	if len(indexes) == 0 {
		return nil, fmt.Errorf("commoncrawl: no indexes available")
	}

	seen := make(map[string]bool)
	var out []string
	var lastErr error
	queriedAny := false

	for i, apiURL := range indexes {
		if ctx.Err() != nil {
			return out, &PartialResultError{Reason: "commoncrawl: context cancelled"}
		}

		names, err := c.queryIndex(ctx, apiURL, domain)
		if err != nil {
			lastErr = err
			continue
		}
		queriedAny = true

		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}

		if i < len(indexes)-1 {
			select {
			case <-time.After(300 * time.Millisecond):
			case <-ctx.Done():
				return out, &PartialResultError{Reason: "commoncrawl: context cancelled"}
			}
		}
	}

	if !queriedAny && lastErr != nil {
		return nil, fmt.Errorf("commoncrawl: %w", lastErr)
	}
	if lastErr != nil {
		return out, &PartialResultError{Reason: fmt.Sprintf("commoncrawl: one or more indexes failed: %v", lastErr)}
	}

	return out, nil
}

// recentIndexes fetches collinfo.json and returns, at most, one CDX API URL
// per year for the commoncrawlMaxYears most recent years.
func (c *CommonCrawl) recentIndexes(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, commoncrawlCollinfoURL, nil)
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
		return nil, fmt.Errorf("collinfo.json: unexpected status %d", resp.StatusCode)
	}

	var all []commoncrawlIndex
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("collinfo.json: decode error: %w", err)
	}

	currentYear := time.Now().Year()
	var selected []string
	for y := 0; y < commoncrawlMaxYears; y++ {
		yearStr := strconv.Itoa(currentYear - y)
		for _, idx := range all {
			if strings.Contains(idx.ID, yearStr) && idx.APIURL != "" {
				selected = append(selected, idx.APIURL)
				break
			}
		}
	}
	return selected, nil
}

// queryIndex streams the CDX response for one index rather than buffering
// the whole (potentially large) body in memory.
func (c *CommonCrawl) queryIndex(ctx context.Context, apiURL, domain string) ([]string, error) {
	reqURL := fmt.Sprintf("%s?url=*.%s&output=json&limit=%d", apiURL, domain, commoncrawlPerIndexLimit)
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

	// A 404 from a CDX index means "no captures found" — not an error.
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: unexpected status %d", apiURL, resp.StatusCode)
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var row struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(line, &row); err != nil {
			continue
		}
		u, err := url.Parse(row.URL)
		if err != nil {
			continue
		}
		host := cleanDomain(u.Hostname())
		if host == "" {
			continue
		}
		if host != domain && !strings.HasSuffix(host, suffix) {
			continue
		}
		if !seen[host] {
			seen[host] = true
			out = append(out, host)
		}
	}
	if err := scanner.Err(); err != nil {
		return out, err
	}

	return out, nil
}
