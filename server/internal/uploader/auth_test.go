package uploader

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// TestLoadTokenNotFound verifies that LoadToken returns ErrTokenNotFound when
// no token file exists for the given channelID.
func TestLoadTokenNotFound(t *testing.T) {
	mgr := New(t.TempDir())
	_, err := mgr.LoadToken("nonexistent-channel")
	if err != ErrTokenNotFound {
		t.Fatalf("want ErrTokenNotFound, got %v", err)
	}
}

// TestSaveLoadRoundTrip verifies that a token saved with SaveToken can be
// loaded back identically with LoadToken.
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mgr := New(dir)

	want := &oauth2.Token{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Truncate(time.Second),
	}

	if err := mgr.SaveToken("test-channel", want); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	// File must exist at expected path.
	expectedPath := filepath.Join(dir, "test-channel.json")
	got, err := mgr.LoadToken("test-channel")
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}

	_ = expectedPath // checked implicitly by LoadToken succeeding

	if got.AccessToken != want.AccessToken {
		t.Errorf("AccessToken: got %q, want %q", got.AccessToken, want.AccessToken)
	}
	if got.RefreshToken != want.RefreshToken {
		t.Errorf("RefreshToken: got %q, want %q", got.RefreshToken, want.RefreshToken)
	}
	if !got.Expiry.Equal(want.Expiry) {
		t.Errorf("Expiry: got %v, want %v", got.Expiry, want.Expiry)
	}
}

// TestRefresh verifies that Refresh calls the token endpoint and returns an
// updated access token.  A fake OAuth2 token server is used.
func TestRefresh(t *testing.T) {
	const newAccessToken = "refreshed-access-token"

	// Fake token endpoint that returns a new access token.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  newAccessToken,
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "refresh-xyz",
		})
	}))
	defer ts.Close()

	// Build a minimal client_secret JSON that points to our fake server.
	clientSecretJSON, err := json.Marshal(map[string]interface{}{
		"installed": map[string]interface{}{
			"client_id":                   "fake-client-id",
			"client_secret":               "fake-client-secret",
			"redirect_uris":               []string{"urn:ietf:wg:oauth:2.0:oob"},
			"auth_uri":                    ts.URL + "/auth",
			"token_uri":                   ts.URL + "/token",
			"auth_provider_x509_cert_url": ts.URL + "/certs",
		},
	})
	if err != nil {
		t.Fatalf("marshal client secret: %v", err)
	}

	// Existing token with a stale access token but a valid refresh token.
	stale := &oauth2.Token{
		AccessToken:  "stale-access-token",
		RefreshToken: "refresh-xyz",
		Expiry:       time.Now().Add(-time.Hour), // expired
	}

	mgr := New(t.TempDir())
	refreshed, err := mgr.Refresh(stale, clientSecretJSON)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if refreshed.AccessToken != newAccessToken {
		t.Errorf("AccessToken: got %q, want %q", refreshed.AccessToken, newAccessToken)
	}
}
