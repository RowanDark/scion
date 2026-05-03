package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

type Facebook struct{}

func (f *Facebook) Name() string        { return "Facebook CT" }
func (f *Facebook) ID() string          { return "facebook" }
func (f *Facebook) NeedsKey() bool      { return true }
func (f *Facebook) IsAvailable() bool   { return os.Getenv("FB_APP_ID") != "" && os.Getenv("FB_APP_SECRET") != "" }
func (f *Facebook) DefaultTimeout() int { return 0 }

func (f *Facebook) Run(ctx context.Context, domain string) ([]string, error) {
	appID := os.Getenv("FB_APP_ID")
	appSecret := os.Getenv("FB_APP_SECRET")
	if appID == "" || appSecret == "" {
		return nil, errors.New("facebook: FB_APP_ID and FB_APP_SECRET not set")
	}

	token, err := f.getToken(ctx, appID, appSecret)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var out []string
	after := ""

	for {
		apiURL := fmt.Sprintf("https://graph.facebook.com/certificates?query=%s&fields=domains&access_token=%s&limit=100",
			url.QueryEscape(domain), token)
		if after != "" {
			apiURL += "&after=" + after
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return out, err
		}
		req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return out, err
		}

		var page struct {
			Data []struct {
				Domains []string `json:"domains"`
			} `json:"data"`
			Paging struct {
				Cursors struct {
					After string `json:"after"`
				} `json:"cursors"`
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return out, fmt.Errorf("facebook: decode error: %w", err)
		}
		resp.Body.Close()

		for _, cert := range page.Data {
			for _, d := range cert.Domains {
				name := cleanDomain(d)
				if name != "" && !seen[name] {
					seen[name] = true
					out = append(out, name)
				}
			}
		}

		if page.Paging.Next == "" {
			break
		}
		after = page.Paging.Cursors.After
	}
	return out, nil
}

func (f *Facebook) getToken(ctx context.Context, appID, appSecret string) (string, error) {
	tokenURL := fmt.Sprintf("https://graph.facebook.com/oauth/access_token?client_id=%s&client_secret=%s&grant_type=client_credentials",
		appID, appSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "scion/1.0 (github.com/RowanDark/scion)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("facebook: token decode error: %w", err)
	}
	if result.AccessToken == "" {
		return "", errors.New("facebook: empty access token returned")
	}
	return result.AccessToken, nil
}
