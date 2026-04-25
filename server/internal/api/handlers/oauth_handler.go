package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	internaloauth "github.com/joaoGMPereira/autocut/server/internal/oauth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OAuthHandler handles OAuth client secret profile CRUD and channel authorization flows.
type OAuthHandler struct {
	repo        *database.OAuthClientSecretRepo
	channelRepo *database.ChannelRepo
	sessions    *internaloauth.SessionManager
}

// NewOAuthHandler creates an OAuthHandler.
func NewOAuthHandler(db *sql.DB, channelRepo *database.ChannelRepo, sessions *internaloauth.SessionManager) *OAuthHandler {
	return &OAuthHandler{
		repo:        database.NewOAuthClientSecretRepo(db, slog.Default()),
		channelRepo: channelRepo,
		sessions:    sessions,
	}
}

// oauthProfileResponse is the JSON shape for an OAuth profile.
type oauthProfileResponse struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	ProjectID      string `json:"project_id"`
	IsDefault      bool   `json:"is_default"`
	HasCredentials bool   `json:"has_credentials"` // false when client_id is empty (legacy migration)
}

func toProfileResponse(o database.OAuthClientSecret) oauthProfileResponse {
	return oauthProfileResponse{
		ID:             o.ID,
		Name:           o.Name,
		ProjectID:      o.ProjectID,
		IsDefault:      o.IsDefault,
		HasCredentials: o.ClientID != "",
	}
}

// GetOAuthProfiles returns all OAuth profiles.
// GET /api/oauth/profiles → 200 {profiles: [...]}
func (h *OAuthHandler) GetOAuthProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.repo.List(r.Context())
	if err != nil {
		slog.Error("list oauth profiles failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	out := make([]oauthProfileResponse, len(profiles))
	for i, p := range profiles {
		out[i] = toProfileResponse(p)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"profiles": out})
}

// PostOAuthProfile uploads a new OAuth client secret JSON file.
// POST /api/oauth/profiles — multipart: name (string) + file (.json)
func (h *OAuthHandler) PostOAuthProfile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	// Parse the Google OAuth JSON to extract project_id, client_id, client_secret.
	var oauthJSON struct {
		Installed *struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			ProjectID    string `json:"project_id"`
		} `json:"installed"`
		Web *struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			ProjectID    string `json:"project_id"`
		} `json:"web"`
	}
	if err := json.Unmarshal(data, &oauthJSON); err != nil {
		http.Error(w, "invalid OAuth JSON", http.StatusBadRequest)
		return
	}

	var clientID, clientSecret, projectID string
	if oauthJSON.Installed != nil {
		clientID = oauthJSON.Installed.ClientID
		clientSecret = oauthJSON.Installed.ClientSecret
		projectID = oauthJSON.Installed.ProjectID
	} else if oauthJSON.Web != nil {
		clientID = oauthJSON.Web.ClientID
		clientSecret = oauthJSON.Web.ClientSecret
		projectID = oauthJSON.Web.ProjectID
	} else {
		http.Error(w, "unrecognised OAuth JSON format", http.StatusBadRequest)
		return
	}

	if clientID == "" || clientSecret == "" {
		slog.Warn("oauth profile upload: missing client_id or client_secret in JSON")
		http.Error(w, "JSON is missing client_id or client_secret", http.StatusBadRequest)
		return
	}

	o := &database.OAuthClientSecret{
		Name:         name,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		ProjectID:    projectID,
	}

	id, err := h.repo.Create(r.Context(), o)
	if err != nil {
		slog.Error("create oauth profile failed", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	created, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		slog.Error("get oauth profile failed", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("oauth profile created", "id", id, "name", name)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toProfileResponse(*created))
}

// PatchOAuthProfile re-uploads the OAuth JSON for an existing profile (updates client_id,
// client_secret, project_id in-place without changing the ID or breaking channel links).
// PATCH /api/oauth/profiles/{id} — multipart: file (.json)
func (h *OAuthHandler) PatchOAuthProfile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	existing, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "oauth profile not found", http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	var oauthJSON struct {
		Installed *struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			ProjectID    string `json:"project_id"`
		} `json:"installed"`
		Web *struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			ProjectID    string `json:"project_id"`
		} `json:"web"`
	}
	if err := json.Unmarshal(data, &oauthJSON); err != nil {
		http.Error(w, "invalid OAuth JSON", http.StatusBadRequest)
		return
	}

	var clientID, clientSecret, projectID string
	if oauthJSON.Installed != nil {
		clientID = oauthJSON.Installed.ClientID
		clientSecret = oauthJSON.Installed.ClientSecret
		projectID = oauthJSON.Installed.ProjectID
	} else if oauthJSON.Web != nil {
		clientID = oauthJSON.Web.ClientID
		clientSecret = oauthJSON.Web.ClientSecret
		projectID = oauthJSON.Web.ProjectID
	} else {
		http.Error(w, "unrecognised OAuth JSON format", http.StatusBadRequest)
		return
	}

	if clientID == "" || clientSecret == "" {
		http.Error(w, "JSON is missing client_id or client_secret", http.StatusBadRequest)
		return
	}

	existing.ClientID = clientID
	existing.ClientSecret = clientSecret
	existing.ProjectID = projectID

	if err := h.repo.Update(r.Context(), existing); err != nil {
		slog.Error("update oauth profile failed", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("oauth profile updated", "id", id, "name", existing.Name)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toProfileResponse(*existing))
}

// DeleteOAuthProfile deletes an OAuth profile by id.
// DELETE /api/oauth/profiles/{id} → 204
func (h *OAuthHandler) DeleteOAuthProfile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		slog.Error("delete oauth profile failed", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("oauth profile deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// PostSetDefaultProfile sets an OAuth profile as the default.
// POST /api/oauth/profiles/{id}/set-default → 200 {id, is_default}
func (h *OAuthHandler) PostSetDefaultProfile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.repo.SetDefault(r.Context(), id); err != nil {
		slog.Error("set default oauth profile failed", "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("oauth default profile set", "id", id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "is_default": true})
}

// InitOAuthFlow starts a loopback OAuth flow for a channel.
// POST /api/channels/{id}/auth → 200 {"auth_url": string}
func (h *OAuthHandler) InitOAuthFlow(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	channel, err := h.channelRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}

	if !channel.OAuthClientSecretID.Valid {
		http.Error(w, "channel has no linked OAuth profile", http.StatusBadRequest)
		return
	}

	profile, err := h.repo.GetByID(r.Context(), channel.OAuthClientSecretID.Int64)
	if err != nil {
		http.Error(w, "oauth profile not found", http.StatusBadRequest)
		return
	}

	if profile.ClientID == "" {
		slog.Error("oauth profile has empty client_id", "profile_id", profile.ID, "profile_name", profile.Name, "channel_id", id)
		http.Error(w, "oauth profile has empty client_id — please delete and re-upload the OAuth JSON file", http.StatusBadRequest)
		return
	}

	cfg := &oauth2.Config{
		ClientID:     profile.ClientID,
		ClientSecret: profile.ClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/youtube.upload",
			"https://www.googleapis.com/auth/youtube",
		},
		Endpoint: google.Endpoint,
	}

	authURL, err := h.sessions.StartSession(id, cfg)
	if err != nil {
		if errors.Is(err, internaloauth.ErrSessionActive) {
			http.Error(w, "authorization already in progress", http.StatusConflict)
			return
		}
		slog.Error("start oauth session failed", "channel_id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("oauth flow initiated", "channel_id", id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"auth_url": authURL})
}

// RefreshToken refreshes the OAuth access token for a channel using its stored refresh token.
// POST /api/channels/{id}/auth/refresh → 200 {"expires_at": int64}
func (h *OAuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	channel, err := h.channelRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "channel not found", http.StatusNotFound)
		return
	}

	if !channel.OAuthClientSecretID.Valid || channel.RefreshToken == "" {
		http.Error(w, "channel has no refresh token", http.StatusBadRequest)
		return
	}

	profile, err := h.repo.GetByID(r.Context(), channel.OAuthClientSecretID.Int64)
	if err != nil {
		http.Error(w, "oauth profile not found", http.StatusBadRequest)
		return
	}

	cfg := &oauth2.Config{
		ClientID:     profile.ClientID,
		ClientSecret: profile.ClientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/youtube.upload",
			"https://www.googleapis.com/auth/youtube",
		},
		Endpoint: google.Endpoint,
	}

	ts := cfg.TokenSource(r.Context(), &oauth2.Token{RefreshToken: channel.RefreshToken})
	newToken, err := ts.Token()
	if err != nil {
		slog.Error("token refresh failed", "channel_id", id, "err", err)
		http.Error(w, "token refresh failed", http.StatusInternalServerError)
		return
	}

	if err := h.channelRepo.UpdateTokens(r.Context(), id,
		newToken.AccessToken, newToken.RefreshToken, newToken.Expiry.UnixMilli()); err != nil {
		slog.Error("save refreshed tokens failed", "channel_id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info("token refreshed", "channel_id", id, "expires_at", newToken.Expiry.UnixMilli())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int64{"expires_at": newToken.Expiry.UnixMilli()})
}
