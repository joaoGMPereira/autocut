package uploader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

// ErrTokenNotFound is returned by LoadToken when no token file exists for a channel.
var ErrTokenNotFound = errors.New("token not found")

// OAuthManager handles OAuth 2.0 token lifecycle for multi-channel YouTube accounts.
// Tokens are persisted as JSON files in tokenDir (default: ~/.autocut/tokens/).
// Kotlin ref: OAuthAuthenticator + AuthenticationDelegate
type OAuthManager struct {
	tokenDir string
	log      *slog.Logger
}

// New creates an OAuthManager.
// If tokenDir is empty, it defaults to ~/.autocut/tokens.
func New(tokenDir string) *OAuthManager {
	if tokenDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		tokenDir = filepath.Join(home, ".autocut", "tokens")
	}
	return &OAuthManager{
		tokenDir: tokenDir,
		log:      slog.With("component", "uploader.auth"),
	}
}

// Authorize runs the OAuth2 web-server flow for a channel.
// It opens the browser, starts a local callback server, and exchanges the code for a token.
// Kotlin ref: OAuthAuthenticator — authorization code flow
func (m *OAuthManager) Authorize(clientSecretJSON []byte, channelID string) (*oauth2.Token, error) {
	cfg, err := google.ConfigFromJSON(clientSecretJSON,
		youtube.YoutubeUploadScope,
		youtube.YoutubeForceSslScope,
	)
	if err != nil {
		return nil, fmt.Errorf("parse client secret: %w", err)
	}

	// Pick a random free port for the local callback server.
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("find free port: %w", err)
	}
	cfg.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", port)

	state := fmt.Sprintf("%d", rand.Int63())
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	m.log.Info("opening browser for OAuth authorization — open this URL if browser does not launch",
		"channelID", channelID, "url", authURL)
	_ = openBrowser(authURL)

	// Start local HTTP server to receive the callback code.
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("state"); got != state {
			errCh <- fmt.Errorf("state mismatch: got %q", got)
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- errors.New("no code in callback")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "Authorization successful! You can close this tab.")
		codeCh <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("callback server: %w", err)
		}
	}()

	// Wait for code or error (60s timeout).
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		_ = srv.Shutdown(context.Background())
		return nil, err
	case <-ctx.Done():
		_ = srv.Shutdown(context.Background())
		return nil, errors.New("authorization timed out (60s)")
	}

	_ = srv.Shutdown(context.Background())

	token, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	m.log.Info("authorization successful", "channelID", channelID)
	return token, nil
}

// Refresh uses the stored refresh token to obtain a fresh access token.
// Kotlin ref: OAuthAuthenticator.refreshToken
func (m *OAuthManager) Refresh(token *oauth2.Token, clientSecretJSON []byte) (*oauth2.Token, error) {
	cfg, err := google.ConfigFromJSON(clientSecretJSON,
		youtube.YoutubeUploadScope,
		youtube.YoutubeForceSslScope,
	)
	if err != nil {
		return nil, fmt.Errorf("parse client secret: %w", err)
	}

	ts := cfg.TokenSource(context.Background(), token)
	refreshed, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	m.log.Info("token refreshed successfully")
	return refreshed, nil
}

// LoadToken reads the token for channelID from disk.
// Returns ErrTokenNotFound if the file does not exist.
// Kotlin ref: OAuthAuthenticator — credential loading
func (m *OAuthManager) LoadToken(channelID string) (*oauth2.Token, error) {
	path := m.tokenPath(channelID)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("read token file: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}
	return &token, nil
}

// SaveToken persists the token for channelID to disk as JSON.
// Kotlin ref: OAuthAuthenticator — credential persistence
func (m *OAuthManager) SaveToken(channelID string, token *oauth2.Token) error {
	if err := os.MkdirAll(m.tokenDir, 0o700); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	path := m.tokenPath(channelID)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}

	m.log.Info("token saved", "channelID", channelID, "path", path)
	return nil
}

// tokenPath returns the expected file path for a channelID's token.
func (m *OAuthManager) tokenPath(channelID string) string {
	return filepath.Join(m.tokenDir, channelID+".json")
}

// openBrowser opens url in the default browser (macOS: open, Linux: xdg-open).
func openBrowser(url string) error {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}
	return exec.Command(cmd, url).Start()
}

// freePort picks an available TCP port on localhost.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
