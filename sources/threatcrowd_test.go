package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestThreatCrowd_Run(t *testing.T) {
	const fixture = `{"response_code":"1","subdomains":["www.example.com","dev.example.com","www.example.com"],"undercount":"0"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("domain"); got != "example.com" {
			t.Errorf("expected domain=example.com, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	orig := threatcrowdAPIURL
	threatcrowdAPIURL = server.URL
	defer func() { threatcrowdAPIURL = orig }()

	src := &ThreatCrowd{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 deduped results, got %v", got)
	}
}

func TestThreatCrowd_DefunctEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response_code":"0","subdomains":[],"undercount":"0"}`))
	}))
	defer server.Close()

	orig := threatcrowdAPIURL
	threatcrowdAPIURL = server.URL
	defer func() { threatcrowdAPIURL = orig }()

	src := &ThreatCrowd{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no results, got %v", got)
	}
}
