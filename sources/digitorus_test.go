package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDigitorus_Run(t *testing.T) {
	const fixture = `<html><body>
	<a href="/sub.example.com">sub.example.com</a>
	<a href="/api.example.com">api.example.com</a>
	<p>unrelated-other.com should not match</p>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	orig := digitorusBaseURL
	digitorusBaseURL = server.URL
	defer func() { digitorusBaseURL = orig }()

	src := &Digitorus{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := map[string]bool{"sub.example.com": true, "api.example.com": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want keys %v", got, want)
	}
	for _, d := range got {
		if !want[d] {
			t.Errorf("unexpected result %q", d)
		}
	}
}

func TestDigitorus_404TreatedAsData(t *testing.T) {
	const fixture = `<html><body>404 - but still has sub.example.com listed</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	orig := digitorusBaseURL
	digitorusBaseURL = server.URL
	defer func() { digitorusBaseURL = orig }()

	src := &Digitorus{}
	got, err := src.Run(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(got) != 1 || got[0] != "sub.example.com" {
		t.Fatalf("expected [sub.example.com] from 404 body, got %v", got)
	}
}

func TestDigitorus_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	orig := digitorusBaseURL
	digitorusBaseURL = server.URL
	defer func() { digitorusBaseURL = orig }()

	src := &Digitorus{}
	if _, err := src.Run(context.Background(), "example.com"); err == nil {
		t.Fatal("expected error for 500 status")
	}
}
