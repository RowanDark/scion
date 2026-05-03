package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type VirusTotal struct{}

func (v *VirusTotal) Name() string        { return "VirusTotal" }
func (v *VirusTotal) ID() string          { return "virustotal" }
func (v *VirusTotal) NeedsKey() bool      { return true }
func (v *VirusTotal) IsAvailable() bool   { return os.Getenv("VT_API_KEY") != "" }
func (v *VirusTotal) DefaultTimeout() int { return 0 }

func (v *VirusTotal) Run(ctx context.Context, domain string) ([]string, error) {
	key := os.Getenv("VT_API_KEY")
	if key == "" {
		return nil, errors.New("virustotal: VT_API_KEY not set")
	}

	seen := make(map[string]bool)
	var out []string
	cursor := ""

	for count := 0; count < 500; {
		url := fmt.Sprintf("https://www.virustotal.com/api/v3/domains/%s/subdomains?limit=40", domain)
		if cursor != "" {
			url += "&cursor=" + cursor
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return out, err
		}
		req.Header.Set("x-apikey", key)
		req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return out, err
		}

		var page struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			Meta struct {
				Cursor string `json:"cursor"`
			} `json:"meta"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return out, fmt.Errorf("virustotal: decode error: %w", err)
		}
		resp.Body.Close()

		for _, d := range page.Data {
			name := cleanDomain(d.ID)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
		count += len(page.Data)

		if page.Meta.Cursor == "" || len(page.Data) == 0 {
			break
		}
		cursor = page.Meta.Cursor
		time.Sleep(250 * time.Millisecond)
	}
	return out, nil
}
