package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const tokenURL = "https://auth.opensky-network.org/auth/realms/opensky-network/protocol/openid-connect/token"

// TokenManager fetches and caches an OAuth2 access token using the
// client-credentials flow. OpenSky tokens expire after ~30 minutes, so we
// keep track of expiry and transparently refresh when a caller asks for a
// token that's stale. This is the standard pattern for any OAuth2
// client-credentials API — the same shape you'd use for most enterprise
// SaaS APIs, not just OpenSky.
type TokenManager struct {
	clientID     string
	clientSecret string

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func NewTokenManager(clientID, clientSecret string) *TokenManager {
	return &TokenManager{clientID: clientID, clientSecret: clientSecret}
}

// Enabled reports whether credentials were actually configured. When they
// aren't, we fall back to (heavily rate-limited) anonymous access.
func (tm *TokenManager) Enabled() bool {
	return tm.clientID != "" && tm.clientSecret != ""
}

func (tm *TokenManager) Token() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Refresh a little early (60s buffer) rather than waiting until the
	// exact expiry, so an in-flight request never gets caught by a token
	// that dies mid-air.
	if tm.token != "" && time.Now().Before(tm.expiresAt.Add(-60*time.Second)) {
		return tm.token, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", tm.clientID)
	form.Set("client_secret", tm.clientSecret)

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed: %s", resp.Status)
	}

	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("token response missing access_token")
	}

	tm.token = parsed.AccessToken
	tm.expiresAt = time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	return tm.token, nil
}
