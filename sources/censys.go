package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type CensysSource struct{}

func (s *CensysSource) Name() string        { return "Censys" }
func (s *CensysSource) ID() string          { return "censys" }
func (s *CensysSource) NeedsKey() bool      { return true }
func (s *CensysSource) DefaultTimeout() int { return 60 }
func (s *CensysSource) IsAvailable() bool {
	return os.Getenv("CENSYS_API_ID") != "" && os.Getenv("CENSYS_API_SECRET") != ""
}

func (s *CensysSource) Run(ctx context.Context, domain string) ([]string, error) {
	apiID := os.Getenv("CENSYS_API_ID")
	apiSecret := os.Getenv("CENSYS_API_SECRET")

	seen := make(map[string]bool)
	var out []string
	cursor := ""

	for len(out) < 1000 {
		body := map[string]interface{}{
			"q":        fmt.Sprintf("parsed.names: %s", domain),
			"per_page": 100,
			"fields":   []string{"parsed.names"},
		}
		if cursor != "" {
			body["cursor"] = cursor
		}

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return out, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			"https://search.censys.io/api/v2/certificates/search",
			bytes.NewReader(bodyBytes))
		if err != nil {
			return out, err
		}
		req.SetBasicAuth(apiID, apiSecret)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return out, err
		}

		var page struct {
			Result struct {
				Hits []struct {
					Parsed struct {
						Names []string `json:"names"`
					} `json:"parsed"`
				} `json:"hits"`
				Links struct {
					Next string `json:"next"`
				} `json:"links"`
			} `json:"result"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return out, fmt.Errorf("censys: decode error: %w", err)
		}
		resp.Body.Close()

		for _, hit := range page.Result.Hits {
			for _, name := range hit.Parsed.Names {
				name = cleanDomain(name)
				if name != "" && !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
		}

		if page.Result.Links.Next == "" || len(page.Result.Hits) == 0 {
			break
		}
		cursor = page.Result.Links.Next

		time.Sleep(500 * time.Millisecond)
	}

	return out, nil
}
