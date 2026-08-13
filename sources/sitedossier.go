package sources

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SiteDossier scrapes sitedossier.com's paginated parentdomain listing. The
// site captchas aggressively under load, so requests are throttled well
// below the global concurrency/rate settings and a captcha response is
// treated as a clean stopping point rather than a hard failure.
type SiteDossier struct{}

func (s *SiteDossier) Name() string        { return "SiteDossier" }
func (s *SiteDossier) ID() string          { return "sitedossier" }
func (s *SiteDossier) NeedsKey() bool      { return false }
func (s *SiteDossier) IsAvailable() bool   { return true }
func (s *SiteDossier) DefaultTimeout() int { return 180 }

var sitedossierBaseURL = "https://www.sitedossier.com"

// sitedossierMaxPages bounds pagination depth.
const sitedossierMaxPages = 15

// sitedossierInterval keeps requests to roughly 8/min, per sitedossier's
// aggressive captcha threshold. Var (not const) so tests can shrink it.
var sitedossierInterval = 8 * time.Second

var sitedossierNextRe = regexp.MustCompile(`href="/parentdomain/[^"/]+/(\d+)"`)

func (s *SiteDossier) Run(ctx context.Context, domain string) ([]string, error) {
	pattern := regexp.MustCompile(`(?i)[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?\.` + regexp.QuoteMeta(domain))

	seen := make(map[string]bool)
	var out []string
	offset := 0

	for page := 0; page < sitedossierMaxPages; page++ {
		if page > 0 {
			select {
			case <-time.After(sitedossierInterval):
			case <-ctx.Done():
				return out, &PartialResultError{Reason: "sitedossier: context cancelled"}
			}
		}

		body, nextOffset, err := s.fetchPage(ctx, domain, offset)
		if err != nil {
			if len(out) > 0 {
				return out, &PartialResultError{Reason: fmt.Sprintf("sitedossier: stopped at page %d: %v", page, err)}
			}
			return nil, fmt.Errorf("sitedossier: %w", err)
		}

		if isCaptchaChallenge(body) {
			if len(out) > 0 {
				return out, &PartialResultError{Reason: "sitedossier: captcha challenge encountered"}
			}
			return nil, fmt.Errorf("sitedossier: captcha challenge encountered")
		}

		for _, found := range pattern.FindAllString(body, -1) {
			name := cleanDomain(found)
			if name != "" && !seen[name] {
				seen[name] = true
				out = append(out, name)
			}
		}

		if nextOffset <= offset {
			break
		}
		offset = nextOffset
	}

	return out, nil
}

func (s *SiteDossier) fetchPage(ctx context.Context, domain string, offset int) (string, int, error) {
	pageURL := fmt.Sprintf("%s/parentdomain/%s", sitedossierBaseURL, domain)
	if offset > 0 {
		pageURL = fmt.Sprintf("%s/%d", pageURL, offset)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return "", 0, err
	}
	body := sb.String()

	maxOffset := offset
	for _, m := range sitedossierNextRe.FindAllStringSubmatch(body, -1) {
		n, err := strconv.Atoi(m[1])
		if err == nil && n > maxOffset {
			maxOffset = n
		}
	}

	return body, maxOffset, nil
}

func isCaptchaChallenge(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "captcha") || strings.Contains(lower, "verify you are a human")
}
