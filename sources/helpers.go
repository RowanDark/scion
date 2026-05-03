package sources

import "strings"

func cleanDomain(d string) string {
	d = strings.ToLower(strings.TrimSpace(d))
	if len(d) < 2 {
		return ""
	}
	// strip leading wildcard/percent
	if d[0] == '*' || d[0] == '%' {
		d = d[1:]
	}
	if len(d) > 0 && d[0] == '.' {
		d = d[1:]
	}
	// strip trailing dot (DNS absolute form)
	d = strings.TrimSuffix(d, ".")
	return d
}
