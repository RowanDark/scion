package diff

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// fqdnRe matches valid FQDNs for extraction from arbitrary output formats.
var fqdnRe = regexp.MustCompile(`(?i)\b([a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?\.)+[a-z]{2,}\b`)

// LoadPreviousResults reads a previous Scion output file and extracts domains.
func LoadPreviousResults(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	previous := make(map[string]bool)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		matches := fqdnRe.FindAllString(line, -1)
		for _, m := range matches {
			previous[strings.ToLower(m)] = true
		}
	}
	return previous, scanner.Err()
}

// DiffResults partitions current domains into new and existing slices.
func DiffResults(current []string, previous map[string]bool) (newDomains []string, existing []string) {
	for _, d := range current {
		if previous[strings.ToLower(d)] {
			existing = append(existing, d)
		} else {
			newDomains = append(newDomains, d)
		}
	}
	return newDomains, existing
}
