package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"golang.org/x/oauth2"
)

// ErrSessionActive is returned when an OAuth session is already in progress for a channel.
var ErrSessionActive = errors.New("oauth session already active for this channel")

// OAuthSession holds in-flight state for a single authorization attempt.
type OAuthSession struct {
	channelID int64
	config    *oauth2.Config
	state     string
	cancel    context.CancelFunc
}

// SessionManager tracks active OAuth loopback sessions (one per channel).
type SessionManager struct {
	mu          sync.Mutex
	sessions    map[int64]*OAuthSession
	channelRepo *database.ChannelRepo
}

// NewSessionManager creates a SessionManager that persists tokens via channelRepo.
func NewSessionManager(channelRepo *database.ChannelRepo) *SessionManager {
	return &SessionManager{
		sessions:    make(map[int64]*OAuthSession),
		channelRepo: channelRepo,
	}
}

// StartSession initiates an OAuth loopback flow for channelID.
// It picks a free port, starts a local callback server (5-min timeout), and returns the auth URL.
// Returns ErrSessionActive if a session is already in progress for this channel.
func (m *SessionManager) StartSession(channelID int64, baseCfg *oauth2.Config) (string, error) {
	m.mu.Lock()
	if _, exists := m.sessions[channelID]; exists {
		m.mu.Unlock()
		return "", ErrSessionActive
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		m.mu.Unlock()
		return "", fmt.Errorf("pick port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	stateBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, stateBytes); err != nil {
		listener.Close()
		m.mu.Unlock()
		return "", fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	cfg := *baseCfg
	cfg.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	session := &OAuthSession{
		channelID: channelID,
		config:    &cfg,
		state:     state,
		cancel:    cancel,
	}
	m.sessions[channelID] = session
	m.mu.Unlock()

	go m.runCallbackServer(ctx, session, listener)

	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	slog.Info("oauth session started", "channel_id", channelID, "port", port)
	return authURL, nil
}

// runCallbackServer serves the OAuth redirect endpoint until the context is done or a token is received.
func (m *SessionManager) runCallbackServer(ctx context.Context, session *OAuthSession, listener net.Listener) {
	defer func() {
		session.cancel()
		m.mu.Lock()
		delete(m.sessions, session.channelID)
		m.mu.Unlock()
		slog.Info("oauth session closed", "channel_id", session.channelID)
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != session.state {
			http.Error(w, "invalid state", http.StatusBadRequest)
			slog.Warn("oauth callback: invalid state", "channel_id", session.channelID)
			return
		}

		errParam := r.URL.Query().Get("error")
		if errParam != "" {
			http.Error(w, "authorization denied: "+errParam, http.StatusBadRequest)
			slog.Warn("oauth callback: authorization denied", "channel_id", session.channelID, "error", errParam)
			go session.cancel()
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			slog.Warn("oauth callback: missing code", "channel_id", session.channelID)
			return
		}

		token, err := session.config.Exchange(ctx, code)
		if err != nil {
			http.Error(w, "token exchange failed", http.StatusInternalServerError)
			slog.Error("oauth token exchange failed", "channel_id", session.channelID, "err", err)
			go session.cancel()
			return
		}

		if err := m.channelRepo.UpdateTokens(r.Context(), session.channelID,
			token.AccessToken, token.RefreshToken, token.Expiry.UnixMilli()); err != nil {
			http.Error(w, "failed to save tokens", http.StatusInternalServerError)
			slog.Error("oauth save tokens failed", "channel_id", session.channelID, "err", err)
			go session.cancel()
			return
		}

		slog.Info("oauth tokens saved", "channel_id", session.channelID, "expires_at", token.Expiry.UnixMilli())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>AutoCut</title></head><body style="font-family:sans-serif;text-align:center;padding:40px"><h1>Authorization successful</h1><p>Return to AutoCut.</p></body></html>`))

		go session.cancel()
	})

	srv := &http.Server{Handler: mux}
	go func() {
		<-ctx.Done()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutCancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("oauth callback server error", "channel_id", session.channelID, "err", err)
	}
}
