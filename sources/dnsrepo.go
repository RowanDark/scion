package sources

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type DNSRepoSource struct{}

func (s *DNSRepoSource) Name() string        { return "DNSRepo" }
func (s *DNSRepoSource) ID() string          { return "dnsrepo" }
func (s *DNSRepoSource) NeedsKey() bool      { return false }
func (s *DNSRepoSource) DefaultTimeout() int { return 45 }
func (s *DNSRepoSource) IsAvailable() bool   { return true }

func (s *DNSRepoSource) Run(ctx context.Context, domain string) ([]string, error) {
	time.Sleep(1 * time.Second)

	url := fmt.Sprintf("https://dnsrepo.noc.org/?domain=%s", domain)
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dnsrepo: unexpected status %d", resp.StatusCode)
	}

	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string

	tokenizer := html.NewTokenizer(resp.Body)
	inTD := false
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return out, nil
		case html.StartTagToken:
			tag, _ := tokenizer.TagName()
			inTD = string(tag) == "td"
		case html.EndTagToken:
			tag, _ := tokenizer.TagName()
			if string(tag) == "td" {
				inTD = false
			}
		case html.TextToken:
			if inTD {
				text := strings.TrimSpace(string(tokenizer.Text()))
				name := cleanDomain(text)
				if name != "" && (name == domain || strings.HasSuffix(name, suffix)) && !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
		}
	}
}
