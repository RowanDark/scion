package filter

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// LoadScope reads a scope file and returns the list of scope entries.
// Lines beginning with # and blank lines are ignored.
func LoadScope(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var scope []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		scope = append(scope, strings.ToLower(line))
	}
	return scope, scanner.Err()
}

// globToRegex converts a DNS glob pattern to a compiled regexp.
// * matches any sequence of non-dot characters within a single label.
// A leading *. is treated as an optional label (zero or one label + dot),
// so *.qa*.example.com also matches qaFOO.example.com (zero leading labels).
func globToRegex(pattern string) (*regexp.Regexp, error) {
	escaped := regexp.QuoteMeta(pattern)
	regexStr := strings.ReplaceAll(escaped, `\*`, `[^.]*`)
	// Leading *. → optional single label, making *.qa*.x.com match qaFOO.x.com
	if strings.HasPrefix(regexStr, `[^.]*\.`) {
		regexStr = `(?:[^.]+\.)?` + regexStr[len(`[^.]*\.`):]
	}
	return regexp.Compile("^" + regexStr + "$")
}

// MatchesScope returns true if domain matches any entry in scope.
// Supported patterns: exact match, simple wildcard (*. prefix), or DNS glob.
func MatchesScope(domain string, scope []string) bool {
	domain = strings.ToLower(domain)
	for _, entry := range scope {
		entry = strings.ToLower(entry)
		if domain == entry {
			return true
		}
		// Simple suffix wildcard: *.example.com → any subdomain
		if strings.HasPrefix(entry, "*.") && !strings.Contains(entry[2:], "*") {
			suffix := entry[1:] // ".example.com"
			if strings.HasSuffix(domain, suffix) {
				return true
			}
			continue
		}
		// Complex glob: convert to regex
		re, err := globToRegex(entry)
		if err != nil {
			continue
		}
		if re.MatchString(domain) {
			return true
		}
	}
	return false
}
