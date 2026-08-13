package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWayback_Run(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[["original"],["https://www.example.com/a"],["https://api.example.com/b"],["https://www.example.com/c"]]`))
	}))
	defer server.Close()

	orig := waybackAPIURL
	waybackAPIURL = server.URL
	defer func() { waybackAPIURL = orig }()

	src := &Wayback{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]bool{"www.example.com": true, "api.example.com": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want keys %v", got, want)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected result %q", d)
		}
	}
}

func TestWayback_EmptyResultSet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	orig := waybackAPIURL
	waybackAPIURL = server.URL
	defer func() { waybackAPIURL = orig }()

	src := &Wayback{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no results, got %v", got)
	}
}

func TestWayback_HTMLThrottleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>Rate limited</body></html>`))
	}))
	defer server.Close()

	orig := waybackAPIURL
	waybackAPIURL = server.URL
	defer func() { waybackAPIURL = orig }()

	src := &Wayback{}
	if _, err := src.Run(context.Background(), "example.com"); err == nil {
		t.Fatal("expected an error when the CDX API returns HTML instead of JSON")
	}
}

func TestWayback_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[["original"],["https://www.example.com/a"`)) // truncated
	}))
	defer server.Close()

	orig := waybackAPIURL
	waybackAPIURL = server.URL
	defer func() { waybackAPIURL = orig }()

	src := &Wayback{}
	if _, err := src.Run(context.Background(), "example.com"); err == nil {
		t.Fatal("expected an error for malformed/truncated JSON")
	}
}
