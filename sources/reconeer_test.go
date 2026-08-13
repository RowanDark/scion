package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestReconeerSource_NoOpWithoutKey(t *testing.T) {
	os.Unsetenv("RECONEER_API_KEY")

	src := &ReconeerSource{}
	if !src.NeedsKey() {
		t.Error("expected NeedsKey() = true")
	}
	if src.IsAvailable() {
		t.Error("expected IsAvailable() = false when RECONEER_API_KEY is unset")
	}

	if _, err := src.Run(context.Background(), "example.com"); err == nil {
		t.Error("expected Run() to error when no key is configured")
	}
}

func TestReconeerSource_Run(t *testing.T) {
	const fixture = `{"subdomains":["www.example.com","api.example.com","evil-other.com"]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-KEY"); got != "test-key" {
			t.Errorf("expected X-API-KEY header test-key, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	origBase := reconeerBaseURL
	reconeerBaseURL = server.URL
	defer func() { reconeerBaseURL = origBase }()

	os.Setenv("RECONEER_API_KEY", "test-key")
	defer os.Unsetenv("RECONEER_API_KEY")

	src := &ReconeerSource{}
	if !src.IsAvailable() {
		t.Fatal("expected IsAvailable() = true when key is set")
	}

	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]bool{"www.example.com": true, "api.example.com": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want keys %v (out-of-scope host should be filtered)", got, want)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected result %q", d)
		}
	}
}

func TestReconeerSource_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	origBase := reconeerBaseURL
	reconeerBaseURL = server.URL
	defer func() { reconeerBaseURL = origBase }()

	os.Setenv("RECONEER_API_KEY", "test-key")
	defer os.Unsetenv("RECONEER_API_KEY")

	src := &ReconeerSource{}
	if _, err := src.Run(context.Background(), "example.com"); err == nil {
		t.Fatal("expected error on 429")
	}
}
