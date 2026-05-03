package sources

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type RapidDNS struct{}

func (r *RapidDNS) Name() string        { return "RapidDNS" }
func (r *RapidDNS) ID() string          { return "rapiddns" }
func (r *RapidDNS) NeedsKey() bool      { return false }
func (r *RapidDNS) IsAvailable() bool   { return true }
func (r *RapidDNS) DefaultTimeout() int { return 0 }

var fqdnRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}$`)

func (r *RapidDNS) Run(ctx context.Context, domain string) ([]string, error) {
	url := fmt.Sprintf("https://rapiddns.io/subdomain/%s?full=1", domain)
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

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rapiddns: html parse error: %w", err)
	}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if fqdnRe.MatchString(text) && strings.HasSuffix(text, "."+domain) {
				name := cleanDomain(text)
				if name != "" && !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return out, nil
}
