package sources

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Wayback struct{}

func (w *Wayback) Name() string        { return "Wayback Machine" }
func (w *Wayback) ID() string          { return "wayback" }
func (w *Wayback) NeedsKey() bool      { return false }
func (w *Wayback) IsAvailable() bool   { return true }
func (w *Wayback) DefaultTimeout() int { return 90 }

var waybackAPIURL = "https://web.archive.org/cdx/search/cdx"

func (w *Wayback) Run(ctx context.Context, domain string) ([]string, error) {
	fromYear := time.Now().Year() - 3
	apiURL := fmt.Sprintf(
		"%s?url=*.%s&output=json&fl=original&collapse=urlkey&limit=10000&from=%d0101",
		waybackAPIURL,
		domain,
		fromYear,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Stream the response instead of buffering the whole (potentially very
	// large) CDX result set in memory.
	br := bufio.NewReaderSize(resp.Body, 64*1024)
	peeked, _ := br.Peek(1)
	if len(peeked) > 0 && peeked[0] == '<' {
		return nil, fmt.Errorf(
			"wayback: received HTML instead of JSON — CDX API may be overloaded or throttling this IP. " +
				"Try again later or use --timeout to extend the deadline.",
		)
	}

	dec := json.NewDecoder(br)
	// The response is a top-level JSON array of rows; consume the opening
	// '[' token so we can Decode() one row at a time below.
	if _, err := dec.Token(); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("wayback: failed to parse response: %w", err)
	}

	seen := make(map[string]bool)
	var out []string
	first := true
	for dec.More() {
		if ctx.Err() != nil {
			return out, &PartialResultError{Reason: "wayback: context cancelled mid-pagination"}
		}

		var row []string
		if err := dec.Decode(&row); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if len(out) > 0 {
					return out, &PartialResultError{Reason: "wayback: context cancelled mid-pagination"}
				}
				return nil, err
			}
			if len(out) > 0 {
				return out, &PartialResultError{Reason: fmt.Sprintf("wayback: stopped mid-stream: %v", err)}
			}
			return nil, fmt.Errorf("wayback: failed to parse response: %w", err)
		}

		if first {
			// The first row is the CDX header (["original"]), not a result.
			first = false
			continue
		}

		if len(row) == 0 {
			continue
		}
		raw := row[0]
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Hostname())
		host = strings.TrimSuffix(host, ".")
		if host != "" && !seen[host] {
			seen[host] = true
			out = append(out, host)
		}
	}
	return out, nil
}
