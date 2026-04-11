package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
	"github.com/joaoGMPereira/autocut/server/internal/hub"
	"github.com/joaoGMPereira/autocut/server/internal/uploader"
	"golang.org/x/oauth2"
)

// remoteVideoCacheTTL is the maximum age of cached remote_videos rows before re-fetching.
const remoteVideoCacheTTL = time.Hour

// massUpdateMaxVideos is the per-run quota guard for YouTube Data API v3.
const massUpdateMaxVideos = 50

// YouTubeHandler handles FP-015 (Mass Update) and FP-016 (Comment Sync) routes.
// OAuth is built per-request from the channel's stored tokens — no mutable state.
type YouTubeHandler struct {
	hub         *hub.SSEHub
	db          *sql.DB
	channelRepo *database.ChannelRepo
	remoteRepo  *database.RemoteVideoRepo
	commentRepo *database.VideoCommentRepo
	log         *slog.Logger
}

// NewYouTubeHandler creates a YouTubeHandler.
func NewYouTubeHandler(h *hub.SSEHub, db *sql.DB) *YouTubeHandler {
	log := slog.Default()
	return &YouTubeHandler{
		hub:         h,
		db:          db,
		channelRepo: database.NewChannelRepo(db, log),
		remoteRepo:  database.NewRemoteVideoRepo(db, log),
		commentRepo: database.NewVideoCommentRepo(db, log),
		log:         slog.With("component", "api", "handler", "youtube"),
	}
}

// --- request / response types -----------------------------------------------

type remoteVideosResponse struct {
	Videos []remoteVideoJSON `json:"videos"`
	Page   int               `json:"page"`
	Total  int               `json:"total"`
	Cached bool              `json:"cached"`
}

type remoteVideoJSON struct {
	YoutubeID   string `json:"youtube_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	CategoryID  int    `json:"category_id"`
	PublishedAt string `json:"published_at,omitempty"`
	FetchedAt   string `json:"fetched_at,omitempty"`
}

type massUpdateFilter struct {
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
	CategoryID *int   `json:"category_id"`
}

type massUpdateUpdates struct {
	TitleTemplate      string   `json:"title_template"`
	DescriptionAppend  string   `json:"description_append"`
	TagsAdd            []string `json:"tags_add"`
	TagsRemove         []string `json:"tags_remove"`
	Hashtags           []string `json:"hashtags"`
	CategoryID         *int     `json:"category_id"`
}

type massUpdateRequest struct {
	ChannelID int64             `json:"channel_id"`
	Filter    massUpdateFilter  `json:"filter"`
	Updates   massUpdateUpdates `json:"updates"`
	DryRun    bool              `json:"dry_run"`
}

type massUpdatePreviewItem struct {
	YoutubeID    string   `json:"youtube_id"`
	OldTitle     string   `json:"old_title"`
	NewTitle     string   `json:"new_title"`
	OldTags      []string `json:"old_tags"`
	NewTags      []string `json:"new_tags"`
	CategoryID   int      `json:"category_id"`
}

type commentSyncResponse struct {
	JobID string `json:"job_id"`
}

type commentsListResponse struct {
	Comments []commentJSON `json:"comments"`
	Page     int           `json:"page"`
	Total    int           `json:"total"`
}

type commentJSON struct {
	ID             int64  `json:"id"`
	YoutubeVideoID string `json:"youtube_video_id"`
	CommentID      string `json:"comment_id"`
	Author         string `json:"author"`
	Text           string `json:"text"`
	Likes          int64  `json:"likes"`
	PublishedAt    string `json:"published_at,omitempty"`
	SyncedAt       string `json:"synced_at,omitempty"`
}

// --- FP-015: Remote Videos --------------------------------------------------

// GetRemoteVideos handles GET /api/videos/remote?channel_id={id}&page={n}.
// Returns cached remote_videos rows; refreshes from YouTube API if cache is stale (>1h).
func (h *YouTubeHandler) GetRemoteVideos(w http.ResponseWriter, r *http.Request) {
	channelID, ok := parseIntParam(w, r.URL.Query().Get("channel_id"), "channel_id")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize := 50

	ctx := r.Context()

	fresh, err := h.remoteRepo.IsCacheFresh(ctx, channelID, remoteVideoCacheTTL)
	if err != nil {
		h.log.Error("cache freshness check failed", "err", err, "channelID", channelID)
		writeError(w, http.StatusInternalServerError, "cache_check_failed", err.Error())
		return
	}

	if !fresh {
		if err := h.refreshRemoteVideoCache(ctx, channelID); err != nil {
			h.log.Error("remote video cache refresh failed", "err", err, "channelID", channelID)
			// Non-fatal: serve stale data if any exists; error only if table is empty.
		}
	}

	videos, total, err := h.remoteRepo.ListByChannel(ctx, channelID, page, pageSize)
	if err != nil {
		h.log.Error("list remote videos failed", "err", err, "channelID", channelID)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	resp := remoteVideosResponse{
		Videos: toRemoteVideoJSONSlice(videos),
		Page:   page,
		Total:  total,
		Cached: fresh,
	}
	writeJSON(w, http.StatusOK, resp)
}

// refreshRemoteVideoCache fetches videos from YouTube API and upserts into remote_videos.
func (h *YouTubeHandler) refreshRemoteVideoCache(ctx context.Context, channelID int64) error {
	yt, err := h.buildYouTubeUploader(ctx, channelID)
	if err != nil {
		return err
	}

	var allVideos []database.RemoteVideo
	pageToken := ""
	for {
		infos, next, err := yt.ListVideos(pageToken, massUpdateMaxVideos)
		if err != nil {
			return err
		}
		for _, v := range infos {
			allVideos = append(allVideos, database.RemoteVideo{
				ChannelID:   channelID,
				YoutubeID:   v.YoutubeID,
				Title:       v.Title,
				Description: v.Description,
				Tags:        strings.Join(v.Tags, ","),
				CategoryID:  safeParseInt(v.CategoryID),
				PublishedAt: v.PublishedAt,
			})
		}
		if next == "" {
			break
		}
		pageToken = next
	}

	return h.remoteRepo.UpsertBatch(ctx, channelID, allVideos)
}

// PostMassUpdate handles POST /api/videos/mass-update.
// Returns 202 {job_id}. The goroutine publishes SSE progress events.
func (h *YouTubeHandler) PostMassUpdate(w http.ResponseWriter, r *http.Request) {
	var req massUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.ChannelID == 0 {
		writeError(w, http.StatusBadRequest, "missing_field", "channel_id is required")
		return
	}

	ctx := r.Context()
	channel, err := h.channelRepo.GetByID(ctx, req.ChannelID)
	if err != nil {
		h.log.Error("channel not found for mass update", "err", err, "channelID", req.ChannelID)
		writeError(w, http.StatusBadRequest, "channel_not_found", "channel not found")
		return
	}
	if channel.AccessToken == "" {
		writeError(w, http.StatusBadRequest, "no_oauth_token", "channel has no OAuth token — authenticate first")
		return
	}

	// Build filter for DB query.
	dbFilter := database.RemoteVideoFilter{CategoryID: req.Filter.CategoryID}
	if req.Filter.DateFrom != "" {
		t, err := time.Parse("2006-01-02", req.Filter.DateFrom)
		if err == nil {
			dbFilter.DateFrom = &t
		}
	}
	if req.Filter.DateTo != "" {
		t, err := time.Parse("2006-01-02", req.Filter.DateTo)
		if err == nil {
			dbFilter.DateTo = &t
		}
	}

	videos, err := h.remoteRepo.ListFiltered(ctx, req.ChannelID, dbFilter, massUpdateMaxVideos)
	if err != nil {
		h.log.Error("list filtered videos failed", "err", err, "channelID", req.ChannelID)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	// Dry-run: return preview without calling YouTube API.
	if req.DryRun {
		preview := buildMassUpdatePreview(videos, req.Updates, channel.Name)
		writeJSON(w, http.StatusOK, map[string]any{
			"dry_run":  true,
			"preview":  preview,
			"total":    len(preview),
		})
		return
	}

	jobID := newJobID()
	go func() {
		bgCtx := context.Background()
		yt, err := h.buildYouTubeUploader(bgCtx, req.ChannelID)
		if err != nil {
			h.log.Error("build youtube uploader failed", "err", err, "channelID", req.ChannelID)
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]any{"message": err.Error()}})
			return
		}

		total := len(videos)
		for i, v := range videos {
			newMeta := applyMassUpdateToVideo(v, req.Updates, channel.Name)
			catStr := strconv.Itoa(newMeta.CategoryID)
			meta := uploader.VideoMetadata{
				VideoID:     v.YoutubeID,
				Title:       newMeta.Title,
				Description: newMeta.Description,
				Tags:        newMeta.Tags,
				CategoryID:  catStr,
			}
			if err := yt.SetMetadata(v.YoutubeID, meta); err != nil {
				h.log.Error("set metadata failed", "err", err, "youtubeID", v.YoutubeID)
				h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]any{
					"youtube_id": v.YoutubeID,
					"message":    err.Error(),
					"index":      i + 1,
					"total":      total,
				}})
				// Continue with remaining videos — partial failure is acceptable.
				continue
			}
			h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]any{
				"index":      i + 1,
				"total":      total,
				"youtube_id": v.YoutubeID,
				"new_title":  newMeta.Title,
			}})
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]any{"total": total}})
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID})
}

// GetMassUpdateStream handles GET /api/videos/mass-update/{id}/stream.
func (h *YouTubeHandler) GetMassUpdateStream(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "id is required")
		return
	}
	h.hub.ServeSSE(w, r, id)
}

// --- FP-016: Comment Sync ---------------------------------------------------

// PostCommentSync handles POST /api/channels/{id}/comments/sync.
// Fetches top-level comments for all remote_videos of the channel and persists them.
func (h *YouTubeHandler) PostCommentSync(w http.ResponseWriter, r *http.Request) {
	channelID, ok := parseIntParam(w, r.PathValue("id"), "id")
	if !ok {
		return
	}

	ctx := r.Context()
	channel, err := h.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		h.log.Error("channel not found for comment sync", "err", err, "channelID", channelID)
		writeError(w, http.StatusBadRequest, "channel_not_found", "channel not found")
		return
	}
	if channel.AccessToken == "" {
		writeError(w, http.StatusBadRequest, "no_oauth_token", "channel has no OAuth token — authenticate first")
		return
	}

	// Collect video IDs to sync from remote_videos (up to 50 latest).
	videos, err := h.remoteRepo.ListFiltered(ctx, channelID, database.RemoteVideoFilter{}, massUpdateMaxVideos)
	if err != nil {
		h.log.Error("list remote videos for comment sync failed", "err", err, "channelID", channelID)
		writeError(w, http.StatusInternalServerError, "list_videos_failed", err.Error())
		return
	}
	if len(videos) == 0 {
		writeError(w, http.StatusBadRequest, "no_remote_videos", "no remote videos cached — fetch remote videos first")
		return
	}

	jobID := newJobID()
	go func() {
		bgCtx := context.Background()
		yt, err := h.buildYouTubeUploader(bgCtx, channelID)
		if err != nil {
			h.log.Error("build youtube uploader failed for comment sync", "err", err, "channelID", channelID)
			h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]any{"message": err.Error()}})
			return
		}

		total := len(videos)
		for i, v := range videos {
			comments, err := yt.GetComments(v.YoutubeID)
			if err != nil {
				h.log.Error("get comments failed", "err", err, "youtubeID", v.YoutubeID)
				h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]any{
					"youtube_id": v.YoutubeID,
					"message":    err.Error(),
					"index":      i + 1,
					"total":      total,
				}})
				continue
			}

			dbComments := make([]database.VideoComment, 0, len(comments))
			for _, c := range comments {
				dbComments = append(dbComments, database.VideoComment{
					YoutubeVideoID: v.YoutubeID,
					ChannelID:      channelID,
					CommentID:      c.ID,
					Author:         c.AuthorName,
					Text:           c.Text,
					Likes:          c.LikeCount,
					PublishedAt:    c.PublishedAt,
				})
			}
			if err := h.commentRepo.UpsertBatch(bgCtx, dbComments); err != nil {
				h.log.Error("upsert comments failed", "err", err, "youtubeID", v.YoutubeID)
				h.hub.Publish(jobID, hub.SSEEvent{Type: "error", Data: map[string]any{
					"youtube_id": v.YoutubeID,
					"message":    err.Error(),
				}})
				continue
			}

			h.hub.Publish(jobID, hub.SSEEvent{Type: "progress", Data: map[string]any{
				"index":         i + 1,
				"total":         total,
				"youtube_id":    v.YoutubeID,
				"comments_saved": len(dbComments),
			}})
		}
		h.hub.Publish(jobID, hub.SSEEvent{Type: "done", Data: map[string]any{"total": total}})
	}()

	writeJSON(w, http.StatusAccepted, commentSyncResponse{JobID: jobID})
}

// GetCommentSyncStream handles GET /api/channels/{id}/comments/sync/{job_id}/stream.
func (h *YouTubeHandler) GetCommentSyncStream(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	if jobID == "" {
		writeError(w, http.StatusBadRequest, "missing_param", "job_id is required")
		return
	}
	h.hub.ServeSSE(w, r, jobID)
}

// GetComments handles GET /api/channels/{id}/comments?page={n}&search={text}.
func (h *YouTubeHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	channelID, ok := parseIntParam(w, r.PathValue("id"), "id")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	search := r.URL.Query().Get("search")
	pageSize := 50

	ctx := r.Context()
	comments, total, err := h.commentRepo.ListByChannel(ctx, channelID, page, pageSize, search)
	if err != nil {
		h.log.Error("list comments failed", "err", err, "channelID", channelID)
		writeError(w, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}

	resp := commentsListResponse{
		Comments: toCommentJSONSlice(comments),
		Page:     page,
		Total:    total,
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- OAuth helper -----------------------------------------------------------

// buildYouTubeUploader constructs a YouTubeUploader using the channel's stored OAuth tokens.
// Access token is used as-is; refresh will be handled transparently by the oauth2 transport.
func (h *YouTubeHandler) buildYouTubeUploader(ctx context.Context, channelID int64) (*uploader.YouTubeUploader, error) {
	channel, err := h.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel.AccessToken == "" {
		return nil, &oauthMissingError{channelID: channelID}
	}

	tok := &oauth2.Token{
		AccessToken:  channel.AccessToken,
		RefreshToken: channel.RefreshToken,
		Expiry:       time.UnixMilli(channel.ExpiresAt),
	}
	httpClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(tok))
	yt, err := uploader.NewYouTubeUploader(httpClient)
	if err != nil {
		h.log.Error("new youtube uploader failed", "err", err, "channelID", channelID)
		return nil, err
	}
	return yt, nil
}

// --- Mass update helpers ----------------------------------------------------

type appliedMeta struct {
	Title       string
	Description string
	Tags        []string
	CategoryID  int
}

// applyMassUpdateToVideo computes the new metadata for a video given the update spec.
func applyMassUpdateToVideo(v database.RemoteVideo, u massUpdateUpdates, channelName string) appliedMeta {
	// Title template resolution.
	newTitle := v.Title
	if u.TitleTemplate != "" {
		newTitle = u.TitleTemplate
		newTitle = strings.ReplaceAll(newTitle, "{original}", v.Title)
		newTitle = strings.ReplaceAll(newTitle, "{channel}", channelName)
		dateStr := ""
		if !v.PublishedAt.IsZero() {
			dateStr = v.PublishedAt.Format("2006-01-02")
		}
		newTitle = strings.ReplaceAll(newTitle, "{date}", dateStr)
	}

	// Description append.
	newDesc := v.Description
	if u.DescriptionAppend != "" {
		newDesc += u.DescriptionAppend
	}
	// Append hashtags to description.
	if len(u.Hashtags) > 0 {
		newDesc += "\n" + strings.Join(u.Hashtags, " ")
	}

	// Tag manipulation.
	existingTags := splitTags(v.Tags)
	removeSet := make(map[string]bool, len(u.TagsRemove))
	for _, t := range u.TagsRemove {
		removeSet[strings.ToLower(t)] = true
	}
	var newTags []string
	for _, t := range existingTags {
		if !removeSet[strings.ToLower(t)] {
			newTags = append(newTags, t)
		}
	}
	newTags = append(newTags, u.TagsAdd...)

	// Category override.
	catID := v.CategoryID
	if u.CategoryID != nil {
		catID = *u.CategoryID
	}

	return appliedMeta{
		Title:       newTitle,
		Description: newDesc,
		Tags:        newTags,
		CategoryID:  catID,
	}
}

// buildMassUpdatePreview returns preview items for dry_run mode.
func buildMassUpdatePreview(videos []database.RemoteVideo, u massUpdateUpdates, channelName string) []massUpdatePreviewItem {
	items := make([]massUpdatePreviewItem, 0, len(videos))
	for _, v := range videos {
		applied := applyMassUpdateToVideo(v, u, channelName)
		items = append(items, massUpdatePreviewItem{
			YoutubeID:  v.YoutubeID,
			OldTitle:   v.Title,
			NewTitle:   applied.Title,
			OldTags:    splitTags(v.Tags),
			NewTags:    applied.Tags,
			CategoryID: applied.CategoryID,
		})
	}
	return items
}

// --- Conversion helpers -----------------------------------------------------

func toRemoteVideoJSONSlice(videos []database.RemoteVideo) []remoteVideoJSON {
	out := make([]remoteVideoJSON, 0, len(videos))
	for _, v := range videos {
		j := remoteVideoJSON{
			YoutubeID:   v.YoutubeID,
			Title:       v.Title,
			Description: v.Description,
			Tags:        v.Tags,
			CategoryID:  v.CategoryID,
		}
		if !v.PublishedAt.IsZero() {
			j.PublishedAt = v.PublishedAt.UTC().Format(time.RFC3339)
		}
		if !v.FetchedAt.IsZero() {
			j.FetchedAt = v.FetchedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, j)
	}
	return out
}

func toCommentJSONSlice(comments []database.VideoComment) []commentJSON {
	out := make([]commentJSON, 0, len(comments))
	for _, c := range comments {
		j := commentJSON{
			ID:             c.ID,
			YoutubeVideoID: c.YoutubeVideoID,
			CommentID:      c.CommentID,
			Author:         c.Author,
			Text:           c.Text,
			Likes:          c.Likes,
		}
		if !c.PublishedAt.IsZero() {
			j.PublishedAt = c.PublishedAt.UTC().Format(time.RFC3339)
		}
		if !c.SyncedAt.IsZero() {
			j.SyncedAt = c.SyncedAt.UTC().Format(time.RFC3339)
		}
		out = append(out, j)
	}
	return out
}

// splitTags converts a comma-separated tag string into a slice, trimming spaces.
func splitTags(tags string) []string {
	if tags == "" {
		return []string{}
	}
	parts := strings.Split(tags, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// parseIntParam parses a string path/query param as int64.
// Returns false and writes a 400 error if parsing fails.
func parseIntParam(w http.ResponseWriter, raw, name string) (int64, bool) {
	if raw == "" {
		writeError(w, http.StatusBadRequest, "missing_param", name+" is required")
		return 0, false
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_param", name+" must be an integer")
		return 0, false
	}
	return v, true
}

// safeParseInt converts a numeric string to int, returning 0 on error.
func safeParseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

// oauthMissingError is a typed error for missing OAuth tokens.
type oauthMissingError struct {
	channelID int64
}

func (e *oauthMissingError) Error() string {
	return "channel " + strconv.FormatInt(e.channelID, 10) + " has no OAuth token"
}
