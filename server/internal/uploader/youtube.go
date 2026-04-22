package uploader

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	youtubev3 "google.golang.org/api/youtube/v3"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// YouTubeUploader uploads videos to YouTube using stored OAuth tokens.
type YouTubeUploader struct {
	channelRepo     *database.ChannelRepo
	oauthSecretRepo *database.OAuthClientSecretRepo
	uploadRepo      *database.UploadRepo
	log             *slog.Logger
}

// NewYouTubeUploader constructs a YouTubeUploader.
func NewYouTubeUploader(
	channelRepo *database.ChannelRepo,
	oauthSecretRepo *database.OAuthClientSecretRepo,
	uploadRepo *database.UploadRepo,
) *YouTubeUploader {
	return &YouTubeUploader{
		channelRepo:     channelRepo,
		oauthSecretRepo: oauthSecretRepo,
		uploadRepo:      uploadRepo,
		log:             slog.With("component", "youtube_uploader"),
	}
}

// Upload performs the full YouTube upload for one queued upload record.
// On success calls MarkCompleted; on failure calls MarkError.
func (u *YouTubeUploader) Upload(ctx context.Context, upload database.Upload) error {
	channel, err := u.channelRepo.GetByID(ctx, upload.ChannelID)
	if err != nil {
		return fmt.Errorf("get channel %d: %w", upload.ChannelID, err)
	}
	if channel.RefreshToken == "" {
		return fmt.Errorf("channel %d has no refresh token — re-authorize OAuth", upload.ChannelID)
	}
	if !channel.OAuthClientSecretID.Valid {
		return fmt.Errorf("channel %d has no oauth client secret linked", upload.ChannelID)
	}

	secret, err := u.oauthSecretRepo.GetByID(ctx, channel.OAuthClientSecretID.Int64)
	if err != nil {
		return fmt.Errorf("get oauth secret for channel %d: %w", upload.ChannelID, err)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     secret.ClientID,
		ClientSecret: secret.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{"https://www.googleapis.com/auth/youtube.upload"},
	}
	token := &oauth2.Token{
		AccessToken:  channel.AccessToken,
		RefreshToken: channel.RefreshToken,
		Expiry:       time.UnixMilli(channel.ExpiresAt),
		TokenType:    "Bearer",
	}
	ts := oauthCfg.TokenSource(ctx, token)

	svc, err := youtubev3.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return fmt.Errorf("create youtube service: %w", err)
	}

	var meta VideoMetadata
	if err := json.Unmarshal([]byte(upload.MetadataJSON), &meta); err != nil {
		return fmt.Errorf("parse metadata for upload %d: %w", upload.ID, err)
	}
	privacy := meta.Privacy
	if privacy == "" {
		privacy = "private"
	}

	if !upload.LocalVideoPath.Valid || upload.LocalVideoPath.String == "" {
		return fmt.Errorf("upload %d has no local video path", upload.ID)
	}
	videoFile, err := os.Open(upload.LocalVideoPath.String)
	if err != nil {
		return fmt.Errorf("open video %s: %w", upload.LocalVideoPath.String, err)
	}
	defer videoFile.Close()

	u.log.Debug("raw tags before sanitization", "upload_id", upload.ID, "raw_tags", meta.Tags)
	// Strip tags that duplicate description hashtags — YouTube adds description hashtags
	// as implicit tags, so sending them explicitly too may trigger invalidTags.
	tagsForUpload := stripDescriptionHashtags(meta.Description, meta.Tags)
	sanitized := sanitizeYouTubeTags(tagsForUpload)
	ytVideo := &youtubev3.Video{
		Snippet: &youtubev3.VideoSnippet{
			Title:                meta.Title,
			Description:          meta.Description,
			Tags:                 sanitized,
			CategoryId:           meta.CategoryID,
			DefaultLanguage:      "pt-BR",
			DefaultAudioLanguage: "pt-BR",
		},
		Status: &youtubev3.VideoStatus{
			PrivacyStatus:           privacy,
			SelfDeclaredMadeForKids: false,
		},
	}
	tagBudget := 0
	for i, t := range sanitized {
		tagBudget += len(t) // UTF-8 bytes
		if i > 0 {
			tagBudget += 2 // ", " separator
		}
	}
	if debugBody, err := json.Marshal(ytVideo); err == nil {
		u.log.Debug("youtube request body", "upload_id", upload.ID, "body", string(debugBody))
	}
	u.log.Info("uploading to youtube",
		"upload_id", upload.ID,
		"title", meta.Title,
		"privacy", privacy,
		"tag_count", len(sanitized),
		"tag_budget", tagBudget,
		"tags", sanitized,
	)

	inserted, err := svc.Videos.Insert([]string{"snippet", "status"}, ytVideo).Media(videoFile).Do()
	if err != nil {
		if apiErr, ok := err.(*googleapi.Error); ok {
			u.log.Error("youtube API error detail",
				"upload_id", upload.ID,
				"http_code", apiErr.Code,
				"message", apiErr.Message,
				"errors", apiErr.Errors,
			)
		}
		return fmt.Errorf("youtube videos.insert: %w", err)
	}

	youtubeID := inserted.Id
	youtubeURL := "https://www.youtube.com/watch?v=" + youtubeID
	u.log.Info("video uploaded", "upload_id", upload.ID, "youtube_id", youtubeID)

	// Set thumbnail (non-fatal if it fails)
	if upload.LocalThumbnailPath.Valid && upload.LocalThumbnailPath.String != "" {
		thumbFile, openErr := os.Open(upload.LocalThumbnailPath.String)
		if openErr == nil {
			defer thumbFile.Close()
			if _, thumbErr := svc.Thumbnails.Set(youtubeID).Media(thumbFile).Do(); thumbErr != nil {
				u.log.Warn("thumbnail set failed (non-fatal)", "upload_id", upload.ID, "err", thumbErr)
			} else {
				u.log.Info("thumbnail set", "upload_id", upload.ID, "youtube_id", youtubeID)
			}
		}
	}

	// Persist refreshed token if it changed
	if newToken, tokErr := ts.Token(); tokErr == nil && newToken.AccessToken != channel.AccessToken {
		if updateErr := u.channelRepo.UpdateTokens(ctx, upload.ChannelID,
			newToken.AccessToken, newToken.RefreshToken, newToken.Expiry.UnixMilli()); updateErr != nil {
			u.log.Warn("failed to save refreshed token", "channel_id", upload.ChannelID, "err", updateErr)
		}
	}

	return u.uploadRepo.MarkCompleted(ctx, upload.ID, youtubeID, youtubeURL)
}

// stripDescriptionHashtags removes from tags any entry that is already present as a
// hashtag in the video description (e.g. "#culturapop" → removes "culturapop" from tags).
// YouTube treats description hashtags as implicit tags; sending them explicitly too
// can trigger invalidTags on some accounts.
func stripDescriptionHashtags(description string, tags []string) []string {
	hashtagSet := make(map[string]struct{})
	for _, word := range strings.Fields(description) {
		if strings.HasPrefix(word, "#") && len(word) > 1 {
			normalized := strings.ToLower(strings.TrimLeft(word, "#"))
			// strip trailing punctuation that may follow a hashtag (e.g. "#nerd.")
			normalized = strings.TrimRight(normalized, ".,!?;:")
			if normalized != "" {
				hashtagSet[normalized] = struct{}{}
			}
		}
	}
	if len(hashtagSet) == 0 {
		return tags
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if _, dup := hashtagSet[strings.ToLower(t)]; !dup {
			out = append(out, t)
		}
	}
	return out
}

// sanitizeYouTubeTags enforces YouTube API tag constraints:
//   - strips control characters and chars YouTube rejects: < > "
//   - trims whitespace, drops empty and duplicate tags
//   - truncates individual tags to 100 runes (rune-safe, not byte-slice)
//   - respects the 500-byte total budget (YouTube counts UTF-8 bytes of the comma+space-joined string "tag1, tag2, tag3")
func sanitizeYouTubeTags(tags []string) []string {
	const maxPerTag = 100
	const maxTotal = 500
	seen := make(map[string]struct{})
	var out []string
	budget := 0
	for _, t := range tags {
		t = strings.TrimSpace(strings.Map(func(r rune) rune {
			if unicode.IsControl(r) || r == '<' || r == '>' || r == '"' {
				return -1
			}
			return r
		}, t))
		if t == "" {
			continue
		}
		runes := []rune(t)
		if len(runes) > maxPerTag {
			runes = runes[:maxPerTag]
			t = string(runes)
		}
		lower := strings.ToLower(t)
		if _, dup := seen[lower]; dup {
			continue
		}
		cost := len(t) // UTF-8 byte length — YouTube counts bytes, not runes (accented chars = 2 bytes each)
		if len(out) > 0 {
			cost += 2 // ", " separator — YouTube counts the comma+space-joined string
		}
		if budget+cost > maxTotal {
			break
		}
		seen[lower] = struct{}{}
		out = append(out, t)
		budget += cost
	}
	return out
}
