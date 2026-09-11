package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTextFormatterOmitsUnresolvedMarker(t *testing.T) {
	// Simulates the post-filter result set that runDomain builds when
	// -verify is set: only resolved hosts should ever reach a formatter.
	results := []Result{
		{Domain: "resolves-a.example.com", Source: "crtsh", Resolves: boolPtr(true)},
		{Domain: "resolves-b.example.com", Source: "wayback", Resolves: boolPtr(true)},
	}

	var buf bytes.Buffer
	if err := (&TextFormatter{}).Write(&buf, results, "example.com", time.Now(), Meta{}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "unresolved") {
		t.Errorf("text output contains %q marker, want it fully removed: %q", "unresolved", out)
	}

	want := "resolves-a.example.com\nresolves-b.example.com\n"
	if out != want {
		t.Errorf("text output = %q, want %q", out, want)
	}
}
