package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// OAuthHandler handles OAuth profile management and the OAuth authorization flow.
// FP-010: OAuth Multi-Profile Management
type OAuthHandler struct {
	db        *sql.DB
	log       *slog.Logger
	secretRepo *database.OAuthClientSecretRepo
	channelRepo *database.ChannelRepo
}

// NewOAuthHandler creates an OAuthHandler.
func NewOAuthHandler(db *sql.DB) *OAuthHandler {
	log := slog.With("component", "api", "handler", "oauth")
	return &OAuthHandler{
		db:          db,
		log:         log,
		secretRepo:  database.NewOAuthClientSecretRepo(db, slog.Default()),
		channelRepo: database.NewChannelRepo(db, slog.Default()),
	}
}

// profileResponse is the safe public view of an OAuthClientSecret (no credentials).
type profileResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ProjectID string `json:"project_id"`
	IsDefault bool   `json:"is_default"`
}

func toProfileResponse(o database.OAuthClientSecret) profileResponse {
	return profileResponse{
		ID:        o.ID,
		Name:      o.Name,
		ProjectID: o.ProjectID,
		IsDefault: o.IsDefault,
	}
}

// clientSecretFile mirrors the JSON structure of a Google OAuth client_secret.json file.
type clientSecretFile struct {
	Installed *clientSecretEntry `json:"installed"`
	Web       *clientSecretEntry `json:"web"`
}

type clientSecretEntry struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	ProjectID    string `json:"project_id"`
}

// ListProfiles handles GET /api/oauth/profiles
func (h *OAuthHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.secretRepo.List(r.Context())
	if err != nil {
		h.log.Error("list oauth profiles failed", "err", err)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	resp := make([]profileResponse, len(profiles))
	for i, p := range profiles {
		resp[i] = toProfileResponse(p)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"profiles": resp})
}

// UploadProfile handles POST /api/oauth/profiles (multipart/form-data)
func (h *OAuthHandler) UploadProfile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "parse_failed", "failed to parse multipart form")
		return
	}

	name := r.FormValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "missing_field", "name is required")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_field", "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read_failed", "failed to read file")
		return
	}

	var csf clientSecretFile
	if err := json.Unmarshal(data, &csf); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "file is not valid JSON")
		return
	}

	var entry *clientSecretEntry
	switch {
	case csf.Installed != nil:
		entry = csf.Installed
	case csf.Web != nil:
		entry = csf.Web
	default:
		writeError(w, http.StatusBadRequest, "invalid_format", "file must contain 'installed' or 'web' key")
		return
	}

	if entry.ClientID == "" || entry.ClientSecret == "" {
		writeError(w, http.StatusBadRequest, "missing_credentials", "client_id and client_secret are required")
		return
	}

	o := &database.OAuthClientSecret{
		Name:         name,
		ClientID:     entry.ClientID,
		ClientSecret: entry.ClientSecret,
		ProjectID:    entry.ProjectID,
		IsDefault:    false,
	}

	id, err := h.secretRepo.Create(r.Context(), o)
	if err != nil {
		h.log.Error("create oauth profile failed", "err", err)
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}

	created, err := h.secretRepo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch_failed", err.Error())
		return
	}

	h.log.Info("oauth profile created", "id", id, "name", name)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"profile": toProfileResponse(*created)})
}

// DeleteProfile handles DELETE /api/oauth/profiles/{id}
func (h *OAuthHandler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.secretRepo.Delete(r.Context(), id); err != nil {
		h.log.Error("delete oauth profile failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}

	h.log.Info("oauth profile deleted", "id", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// SetDefaultProfile handles POST /api/oauth/profiles/{id}/set-default
func (h *OAuthHandler) SetDefaultProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.secretRepo.SetDefault(r.Context(), id); err != nil {
		h.log.Error("set default oauth profile failed", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "set_default_failed", err.Error())
		return
	}

	h.log.Info("oauth profile set as default", "id", id)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// InitOAuthFlow handles POST /api/channels/{id}/auth
// Returns the Google OAuth2 authorization URL for the channel's associated profile.
func (h *OAuthHandler) InitOAuthFlow(w http.ResponseWriter, r *http.Request) {
	channelID, ok := parseID(w, r)
	if !ok {
		return
	}

	ch, err := h.channelRepo.GetByID(r.Context(), channelID)
	if err != nil {
		h.log.Error("get channel failed", "id", channelID, "err", err)
		writeError(w, http.StatusNotFound, "channel_not_found", "channel not found")
		return
	}

	if !ch.OAuthClientSecretID.Valid {
		writeError(w, http.StatusBadRequest, "no_oauth_profile", "no oauth profile associated with this channel")
		return
	}

	secret, err := h.secretRepo.GetByID(r.Context(), ch.OAuthClientSecretID.Int64)
	if err != nil {
		h.log.Error("get oauth secret failed", "id", ch.OAuthClientSecretID.Int64, "err", err)
		writeError(w, http.StatusInternalServerError, "profile_not_found", "oauth profile not found")
		return
	}

	params := url.Values{}
	params.Set("client_id", secret.ClientID)
	params.Set("redirect_uri", "urn:ietf:wg:oauth:2.0:oob")
	params.Set("response_type", "code")
	params.Set("scope", "https://www.googleapis.com/auth/youtube https://www.googleapis.com/auth/youtube.upload https://www.googleapis.com/auth/youtube.force-ssl")
	params.Set("access_type", "offline")

	authURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?%s", params.Encode())

	h.log.Info("oauth flow initiated", "channel_id", channelID)
	writeJSON(w, http.StatusOK, map[string]string{"auth_url": authURL})
}

// HandleOAuthCallback handles POST /api/channels/{id}/auth/callback
// Exchanges the authorization code for tokens and persists them.
func (h *OAuthHandler) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	channelID, ok := parseID(w, r)
	if !ok {
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "code is required")
		return
	}

	ch, err := h.channelRepo.GetByID(r.Context(), channelID)
	if err != nil {
		h.log.Error("get channel failed", "id", channelID, "err", err)
		writeError(w, http.StatusNotFound, "channel_not_found", "channel not found")
		return
	}

	if !ch.OAuthClientSecretID.Valid {
		writeError(w, http.StatusBadRequest, "no_oauth_profile", "no oauth profile associated with this channel")
		return
	}

	secret, err := h.secretRepo.GetByID(r.Context(), ch.OAuthClientSecretID.Int64)
	if err != nil {
		h.log.Error("get oauth secret failed", "id", ch.OAuthClientSecretID.Int64, "err", err)
		writeError(w, http.StatusInternalServerError, "profile_not_found", "oauth profile not found")
		return
	}

	tokens, err := exchangeCodeForTokens(secret.ClientID, secret.ClientSecret, req.Code)
	if err != nil {
		h.log.Error("token exchange failed", "channel_id", channelID, "err", err)
		writeError(w, http.StatusBadGateway, "token_exchange_failed", err.Error())
		return
	}

	// expires_in is seconds from now; store as unix milliseconds
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second).UnixMilli()

	if err := h.channelRepo.UpdateTokens(r.Context(), channelID, tokens.AccessToken, tokens.RefreshToken, expiresAt); err != nil {
		h.log.Error("update tokens failed", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	h.log.Info("oauth tokens saved", "channel_id", channelID, "expires_at", expiresAt)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "expires_at": expiresAt})
}

// RefreshToken handles POST /api/channels/{id}/auth/refresh.
// Uses the stored refresh_token to obtain a new access_token and persists it.
func (h *OAuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	channelID, ok := parseID(w, r)
	if !ok {
		return
	}

	ch, err := h.channelRepo.GetByID(r.Context(), channelID)
	if err != nil {
		h.log.Error("get channel failed", "id", channelID, "err", err)
		writeError(w, http.StatusNotFound, "channel_not_found", "channel not found")
		return
	}

	if ch.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "no_refresh_token", "no refresh token stored")
		return
	}

	if !ch.OAuthClientSecretID.Valid {
		writeError(w, http.StatusBadRequest, "no_oauth_profile", "no oauth profile associated with this channel")
		return
	}

	secret, err := h.secretRepo.GetByID(r.Context(), ch.OAuthClientSecretID.Int64)
	if err != nil {
		h.log.Error("get oauth secret failed", "id", ch.OAuthClientSecretID.Int64, "err", err)
		writeError(w, http.StatusInternalServerError, "profile_not_found", "oauth profile not found")
		return
	}

	form := url.Values{}
	form.Set("client_id", secret.ClientID)
	form.Set("client_secret", secret.ClientSecret)
	form.Set("refresh_token", ch.RefreshToken)
	form.Set("grant_type", "refresh_token")

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/token",
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		h.log.Error("build refresh request failed", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "request_build_failed", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.log.Error("refresh token request failed", "channel_id", channelID, "err", err)
		writeError(w, http.StatusBadGateway, "token_refresh_failed", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "read_response_failed", err.Error())
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.log.Error("token refresh endpoint error", "channel_id", channelID, "status", resp.StatusCode, "body", string(body))
		writeError(w, http.StatusBadGateway, "token_refresh_failed", fmt.Sprintf("token endpoint returned %d", resp.StatusCode))
		return
	}

	var tokens tokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		writeError(w, http.StatusBadGateway, "parse_response_failed", err.Error())
		return
	}
	if tokens.AccessToken == "" {
		writeError(w, http.StatusBadGateway, "no_access_token", "no access_token in refresh response")
		return
	}

	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second).UnixMilli()

	if err := h.channelRepo.UpdateTokens(r.Context(), channelID, tokens.AccessToken, ch.RefreshToken, expiresAt); err != nil {
		h.log.Error("update tokens after refresh failed", "channel_id", channelID, "err", err)
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}

	h.log.Info("oauth token refreshed", "channel_id", channelID, "expires_at", expiresAt)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// tokenResponse mirrors the Google OAuth2 token endpoint response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// exchangeCodeForTokens posts to Google's token endpoint and returns the parsed response.
func exchangeCodeForTokens(clientID, clientSecret, code string) (*tokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", "urn:ietf:wg:oauth:2.0:oob")
	form.Set("grant_type", "authorization_code")

	resp, err := http.Post(
		"https://oauth2.googleapis.com/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("http post token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokens tokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if tokens.AccessToken == "" {
		return nil, fmt.Errorf("no access_token in response")
	}

	return &tokens, nil
}

// parseID extracts the {id} path value and parses it as int64.
// Writes an error response and returns false on failure.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := r.PathValue("id")
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", "id must be an integer")
		return 0, false
	}
	return id, true
}
