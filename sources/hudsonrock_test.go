package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHudsonRockSource_Run(t *testing.T) {
	const fixture = `{
		"data": {
			"employees_urls": [
				{"url": "https://portal.example.com/login"},
				{"url": "https://unrelated-other.com/x"}
			],
			"clients_urls": [
				{"url": "https://vpn.example.com/"}
			]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("domain"); got != "example.com" {
			t.Errorf("expected domain=example.com, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	orig := hudsonrockAPIURL
	hudsonrockAPIURL = server.URL
	defer func() { hudsonrockAPIURL = orig }()

	src := &HudsonRockSource{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]bool{"portal.example.com": true, "vpn.example.com": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want keys %v", got, want)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected result %q (out-of-scope host should have been filtered)", d)
		}
	}
}

func TestHudsonRockSource_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	orig := hudsonrockAPIURL
	hudsonrockAPIURL = server.URL
	defer func() { hudsonrockAPIURL = orig }()

	src := &HudsonRockSource{}
	if _, err := src.Run(context.Background(), "example.com"); err == nil {
		t.Fatal("expected error for non-200 status")
	}
}
