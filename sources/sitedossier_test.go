package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSiteDossier_Pagination(t *testing.T) {
	orig := sitedossierInterval
	sitedossierInterval = time.Millisecond
	defer func() { sitedossierInterval = orig }()

	var mux *http.ServeMux
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	mux = http.NewServeMux()
	mux.HandleFunc("/parentdomain/example.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`
			<table>
			<tr><td><a href="/site/www.example.com">www.example.com</a></td></tr>
			<tr><td><a href="/site/api.example.com">api.example.com</a></td></tr>
			</table>
			<a href="/parentdomain/example.com/101"><b>next</b></a>
		`))
	})
	mux.HandleFunc("/parentdomain/example.com/101", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`
			<table>
			<tr><td><a href="/site/mail.example.com">mail.example.com</a></td></tr>
			</table>
		`))
	})

	origBase := sitedossierBaseURL
	sitedossierBaseURL = server.URL
	defer func() { sitedossierBaseURL = origBase }()

	src := &SiteDossier{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]bool{"www.example.com": true, "api.example.com": true, "mail.example.com": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want keys %v", got, want)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected result %q", d)
		}
	}
}

func TestSiteDossier_CaptchaDegradesGracefully(t *testing.T) {
	orig := sitedossierInterval
	sitedossierInterval = time.Millisecond
	defer func() { sitedossierInterval = orig }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>Please complete the CAPTCHA to continue</body></html>`))
	}))
	defer server.Close()

	origBase := sitedossierBaseURL
	sitedossierBaseURL = server.URL
	defer func() { sitedossierBaseURL = origBase }()

	src := &SiteDossier{}
	_, err := src.Run(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected an error when only a captcha page is returned")
	}
}

func TestSiteDossier_CaptchaAfterSomeResultsIsPartial(t *testing.T) {
	orig := sitedossierInterval
	sitedossierInterval = time.Millisecond
	defer func() { sitedossierInterval = orig }()

	var mux *http.ServeMux
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r)
	}))
	defer server.Close()

	mux = http.NewServeMux()
	mux.HandleFunc("/parentdomain/example.com", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`
			<a href="/site/www.example.com">www.example.com</a>
			<a href="/parentdomain/example.com/101"><b>next</b></a>
		`))
	})
	mux.HandleFunc("/parentdomain/example.com/101", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>CAPTCHA required</body></html>`))
	})

	origBase := sitedossierBaseURL
	sitedossierBaseURL = server.URL
	defer func() { sitedossierBaseURL = origBase }()

	src := &SiteDossier{}
	got, err := src.Run(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected a PartialResultError")
	}
	if _, ok := err.(*PartialResultError); !ok {
		t.Fatalf("expected *PartialResultError, got %T: %v", err, err)
	}
	if len(got) != 1 || got[0] != "www.example.com" {
		t.Fatalf("expected partial result [www.example.com], got %v", got)
	}
}
