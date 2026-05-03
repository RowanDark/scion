package filter

import (
	"bufio"
	"os"
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

// MatchesScope returns true if domain matches any entry in scope.
// Entries may be exact ("admin.example.com") or wildcard ("*.example.com").
func MatchesScope(domain string, scope []string) bool {
	domain = strings.ToLower(domain)
	for _, entry := range scope {
		if strings.HasPrefix(entry, "*.") {
			suffix := entry[1:] // ".example.com"
			if strings.HasSuffix(domain, suffix) {
				return true
			}
		} else if domain == entry {
			return true
		}
	}
	return false
}
