package filter

import "testing"

func TestMatchesScope(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		domain  string
		want    bool
	}{
		// Exact match
		{"exact match", "internal-api.paylution.com", "internal-api.paylution.com", true},
		{"exact no match - extra label", "internal-api.paylution.com", "internal-api.qa.paylution.com", false},
		{"exact no match - different", "internal-api.paylution.com", "api.paylution.com", false},

		// Simple wildcard *. prefix
		{"simple wildcard matches subdomain", "*.uat.paylution.com", "api.uat.paylution.com", true},
		{"simple wildcard matches direct sub", "*.paylution.com", "mail.paylution.com", true},
		{"simple wildcard no match - apex", "*.paylution.com", "paylution.com", false},
		{"simple wildcard no match - unrelated", "*.paylution.com", "api.example.com", false},

		// Complex glob: *.qa*.paylution.com
		// Leading *. is optional, so qaFOO.paylution.com matches too.
		{"complex glob - leading label optional, qa prefix", "*.qa*.paylution.com", "qamaster-portal.paylution.com", true},
		{"complex glob - leading label optional, qa prefix 2", "*.qa*.paylution.com", "qarelease-docs.paylution.com", true},
		{"complex glob - with explicit leading label", "*.qa*.paylution.com", "foo.qabar.paylution.com", true},
		{"complex glob - no qa component", "*.qa*.paylution.com", "api.paylution.com", false},
		{"complex glob - qa in suffix only", "*.qa*.paylution.com", "api.qa.paylution.com", true},

		// Case insensitivity
		{"case insensitive domain", "*.paylution.com", "API.PAYLUTION.COM", true},
		{"case insensitive pattern", "*.PAYLUTION.COM", "api.paylution.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesScope(tt.domain, []string{tt.pattern})
			if got != tt.want {
				t.Errorf("MatchesScope(%q, [%q]) = %v, want %v", tt.domain, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestGlobToRegex(t *testing.T) {
	tests := []struct {
		pattern string
		domain  string
		want    bool
	}{
		// Leading optional label
		{"*.qa*.paylution.com", "qamaster-portal.paylution.com", true},
		{"*.qa*.paylution.com", "foo.qabar.paylution.com", true},
		{"*.qa*.paylution.com", "api.paylution.com", false},

		// No leading wildcard
		{"qa*.paylution.com", "qamaster-portal.paylution.com", true},
		{"qa*.paylution.com", "api.paylution.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"/"+tt.domain, func(t *testing.T) {
			re, err := globToRegex(tt.pattern)
			if err != nil {
				t.Fatalf("globToRegex(%q) error: %v", tt.pattern, err)
			}
			got := re.MatchString(tt.domain)
			if got != tt.want {
				t.Errorf("globToRegex(%q).MatchString(%q) = %v, want %v (regex: %s)", tt.pattern, tt.domain, got, tt.want, re)
			}
		})
	}
}
