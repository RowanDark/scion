package sources

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type DNSDumpsterSource struct{}

func (s *DNSDumpsterSource) Name() string        { return "DNSDumpster" }
func (s *DNSDumpsterSource) ID() string          { return "dnsdumpster" }
func (s *DNSDumpsterSource) NeedsKey() bool      { return false }
func (s *DNSDumpsterSource) DefaultTimeout() int { return 45 }
func (s *DNSDumpsterSource) IsAvailable() bool   { return true }

func (s *DNSDumpsterSource) Run(ctx context.Context, domain string) ([]string, error) {
	baseURL := "https://dnsdumpster.com/"

	// Step 1: GET to retrieve CSRF token cookie and hidden form field.
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, err
	}
	getReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0")

	jar := newSimpleCookieJar()
	client := &http.Client{Jar: jar}

	getResp, err := client.Do(getReq)
	if err != nil {
		return nil, err
	}
	defer getResp.Body.Close()

	getBody, err := io.ReadAll(getResp.Body)
	if err != nil {
		return nil, err
	}

	csrfToken := extractCSRFToken(getBody)

	// Fall back to cookie if hidden field not found.
	if csrfToken == "" {
		for _, c := range jar.cookies {
			if c.Name == "csrftoken" {
				csrfToken = c.Value
				break
			}
		}
	}

	if csrfToken == "" {
		// Attempt regex fallback on raw HTML.
		return extractDomainsRegex(string(getBody), domain), nil
	}

	// Step 2: POST with CSRF token and target domain.
	formData := url.Values{}
	formData.Set("csrfmiddlewaretoken", csrfToken)
	formData.Set("targetip", domain)
	formData.Set("user", "free")

	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("Referer", baseURL)
	postReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:124.0) Gecko/20100101 Firefox/124.0")

	// Forward cookies from GET response.
	for _, c := range jar.cookies {
		postReq.AddCookie(c)
	}

	postResp, err := client.Do(postReq)
	if err != nil {
		return nil, err
	}
	defer postResp.Body.Close()

	postBody, err := io.ReadAll(postResp.Body)
	if err != nil {
		return nil, err
	}

	results := extractDomainsHTML(postBody, domain)
	if len(results) == 0 {
		// Regex fallback in case HTML structure changed.
		results = extractDomainsRegex(string(postBody), domain)
	}
	return results, nil
}

// simpleCookieJar is a minimal cookie jar for CSRF handling.
type simpleCookieJar struct {
	cookies []*http.Cookie
}

func newSimpleCookieJar() *simpleCookieJar { return &simpleCookieJar{} }

func (j *simpleCookieJar) SetCookies(_ *url.URL, cookies []*http.Cookie) {
	j.cookies = append(j.cookies, cookies...)
}

func (j *simpleCookieJar) Cookies(_ *url.URL) []*http.Cookie { return j.cookies }

// extractCSRFToken parses the hidden csrfmiddlewaretoken input field from the page HTML.
func extractCSRFToken(body []byte) string {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return ""
	}
	var token string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			var name, value string
			for _, a := range n.Attr {
				if a.Key == "name" {
					name = a.Val
				}
				if a.Key == "value" {
					value = a.Val
				}
			}
			if name == "csrfmiddlewaretoken" {
				token = value
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return token
}

// extractDomainsHTML walks the parsed HTML and collects text nodes matching the target domain.
func extractDomainsHTML(body []byte, domain string) []string {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil
	}
	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if strings.HasSuffix(text, suffix) || text == domain {
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
	return out
}

var dnsDumpsterRe = regexp.MustCompile(`[a-zA-Z0-9][\w\-\.]*\.` + `[a-zA-Z]{2,}`)

// extractDomainsRegex is a fallback that uses a regex on the raw HTML body.
func extractDomainsRegex(body, domain string) []string {
	suffix := "." + domain
	seen := make(map[string]bool)
	var out []string
	for _, match := range dnsDumpsterRe.FindAllString(body, -1) {
		if strings.HasSuffix(match, suffix) || match == domain {
			name := cleanDomain(match)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// Ensure simpleCookieJar satisfies http.CookieJar.
var _ http.CookieJar = (*simpleCookieJar)(nil)

