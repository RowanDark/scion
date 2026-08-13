package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCommonCrawl_Run(t *testing.T) {
	year := time.Now().Year()

	var mux *http.ServeMux
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	mux = http.NewServeMux()
	mux.HandleFunc("/collinfo.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `[
			{"id": "CC-MAIN-%d-33", "cdx-api": "%s/index/current"},
			{"id": "CC-MAIN-%d-01", "cdx-api": "%s/index/prior"}
		]`, year, server.URL, year-1, server.URL)
	})
	mux.HandleFunc("/index/current", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("url"); got != "*.example.com" {
			t.Errorf("expected url=*.example.com, got %q", got)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"url":"https://www.example.com/page1","timestamp":"20240101000000"}`)
		fmt.Fprintln(w, `{"url":"https://api.example.com/v1","timestamp":"20240102000000"}`)
	})
	mux.HandleFunc("/index/prior", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"url":"https://old.example.com/","timestamp":"20230101000000"}`)
	})

	origCollinfo := commoncrawlCollinfoURL
	commoncrawlCollinfoURL = server.URL + "/collinfo.json"
	defer func() { commoncrawlCollinfoURL = origCollinfo }()

	src := &CommonCrawl{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]bool{"www.example.com": true, "api.example.com": true, "old.example.com": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want keys %v", got, want)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected result %q", d)
		}
	}
}

func TestCommonCrawl_OneIndexFailsButOthersSucceed(t *testing.T) {
	year := time.Now().Year()

	var mux *http.ServeMux
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	mux = http.NewServeMux()
	mux.HandleFunc("/collinfo.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `[
			{"id": "CC-MAIN-%d-33", "cdx-api": "%s/index/broken"},
			{"id": "CC-MAIN-%d-01", "cdx-api": "%s/index/ok"}
		]`, year, server.URL, year-1, server.URL)
	})
	mux.HandleFunc("/index/broken", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/index/ok", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"url":"https://sub.example.com/","timestamp":"20230101000000"}`)
	})

	origCollinfo := commoncrawlCollinfoURL
	commoncrawlCollinfoURL = server.URL + "/collinfo.json"
	defer func() { commoncrawlCollinfoURL = origCollinfo }()

	src := &CommonCrawl{}
	got, err := src.Run(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected a PartialResultError since one index failed")
	}
	var partialErr *PartialResultError
	if !isPartialResultError(err, &partialErr) {
		t.Fatalf("expected *PartialResultError, got %T: %v", err, err)
	}
	if len(got) != 1 || got[0] != "sub.example.com" {
		t.Fatalf("expected partial result [sub.example.com], got %v", got)
	}
}

func isPartialResultError(err error, target **PartialResultError) bool {
	pe, ok := err.(*PartialResultError)
	if ok {
		*target = pe
	}
	return ok
}

func TestCommonCrawl_CollinfoUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origCollinfo := commoncrawlCollinfoURL
	commoncrawlCollinfoURL = server.URL
	defer func() { commoncrawlCollinfoURL = origCollinfo }()

	src := &CommonCrawl{}
	if _, err := src.Run(context.Background(), "example.com"); err == nil {
		t.Fatal("expected error when collinfo.json is unavailable")
	} else if !strings.Contains(err.Error(), "commoncrawl") {
		t.Errorf("expected error to be prefixed with commoncrawl:, got %v", err)
	}
}
