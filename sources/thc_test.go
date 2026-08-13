package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTHCSource_Pagination(t *testing.T) {
	pages := map[string]string{
		"":      `{"domains":[{"domain":"www.example.com"},{"domain":"api.example.com"}],"next_page_state":"page2"}`,
		"page2": `{"domains":[{"domain":"mail.example.com"}],"next_page_state":""}`,
	}

	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", ct)
		}
		var body struct {
			Domain    string `json:"domain"`
			PageState string `json:"page_state"`
			Limit     int    `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Domain != "example.com" {
			t.Errorf("expected domain example.com, got %q", body.Domain)
		}
		resp, ok := pages[body.PageState]
		if !ok {
			t.Fatalf("unexpected page_state %q", body.PageState)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	}))
	defer server.Close()

	orig := thcAPIURL
	thcAPIURL = server.URL
	defer func() { thcAPIURL = orig }()

	src := &THCSource{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]bool{"www.example.com": true, "api.example.com": true, "mail.example.com": true}
	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d: %v", len(got), len(want), got)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected result %q", d)
		}
	}
	if requests != 2 {
		t.Errorf("expected 2 requests, got %d", requests)
	}
}

func TestTHCSource_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	orig := thcAPIURL
	thcAPIURL = server.URL
	defer func() { thcAPIURL = orig }()

	src := &THCSource{}
	_, err := src.Run(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
