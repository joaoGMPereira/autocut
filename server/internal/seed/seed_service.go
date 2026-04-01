package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/joaoGMPereira/autocut/server/internal/configurator"
	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// seedConfig mirrors seed_config.json structure (Kotlin: SeedConfig).
type seedConfig struct {
	Channels []seedChannel `json:"channels"`
}

// seedChannel is one entry in seed_config.json (Kotlin: SeedChannel).
type seedChannel struct {
	Name               string `json:"name"`
	ClientSecretFile   string `json:"clientSecretFile"`
	GradientColorStart string `json:"gradientColorStart"`
	GradientColorEnd   string `json:"gradientColorEnd"`
	MadeForKids        bool   `json:"madeForKids"`
}

// clientSecretJSON represents a Google OAuth2 client_secret_*.json file.
type clientSecretJSON struct {
	Installed struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		ProjectID    string `json:"project_id"`
	} `json:"installed"`
}

// SeedService seeds initial channels, channel configs, OAuth secrets,
// and music files into a fresh AutoCut installation.
// Port of SeedService.kt — idempotent: safe to call on every startup.
type SeedService struct {
	db  *sql.DB
	dir *configurator.AutoCutDir
	log *slog.Logger
}

// New creates a SeedService.
func New(db *sql.DB, dir *configurator.AutoCutDir, log *slog.Logger) *SeedService {
	return &SeedService{db: db, dir: dir, log: log.With("service", "seed")}
}

// Run executes the full seed sequence:
//  1. Extracts seed_config.json and client_secret_*.json to ~/.autocut/config/
//  2. Extracts MP3s to ~/.autocut/shared/music/
//  3. For each channel in seed_config: creates OAuthClientSecret, Channel, ChannelConfig
//
// Idempotent: existing records are skipped.
func (s *SeedService) Run(ctx context.Context) error {
	if err := s.extractSeedFiles(); err != nil {
		return fmt.Errorf("extract seed files: %w", err)
	}
	if err := s.extractMusic(); err != nil {
		return fmt.Errorf("extract music: %w", err)
	}

	cfg, err := s.loadSeedConfig()
	if err != nil {
		return fmt.Errorf("load seed_config.json: %w", err)
	}

	s.log.Info("seeding channels", "count", len(cfg.Channels))
	for _, ch := range cfg.Channels {
		if err := s.seedChannel(ctx, ch); err != nil {
			s.log.Warn("seed channel failed (skipping)", "channel", ch.Name, "err", err)
		}
	}
	s.log.Info("seed complete")
	return nil
}

// extractSeedFiles copies embedded seed_config.json and client_secret_*.json
// to ~/.autocut/config/ if they don't already exist there.
func (s *SeedService) extractSeedFiles() error {
	if err := os.MkdirAll(s.dir.ConfigDir, 0755); err != nil {
		return err
	}

	entries, err := assets.ReadDir("assets")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name != "seed_config.json" && !strings.HasPrefix(name, "client_secret_") {
			continue
		}
		dst := filepath.Join(s.dir.ConfigDir, name)
		if _, err := os.Stat(dst); err == nil {
			continue // already exists
		}
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", name, err)
		}
		if err := os.WriteFile(dst, data, 0600); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
		s.log.Info("extracted seed file", "file", name)
	}
	return nil
}

// extractMusic copies embedded MP3s to ~/.autocut/shared/music/ if not present.
func (s *SeedService) extractMusic() error {
	if err := os.MkdirAll(s.dir.SharedMusicDir, 0755); err != nil {
		return err
	}

	entries, err := fs.ReadDir(assets, "assets/music")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		dst := filepath.Join(s.dir.SharedMusicDir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue // already exists
		}
		data, err := assets.ReadFile("assets/music/" + e.Name())
		if err != nil {
			return fmt.Errorf("read embedded music %s: %w", e.Name(), err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("write music %s: %w", dst, err)
		}
		s.log.Info("extracted music file", "file", e.Name())
	}
	return nil
}

// loadSeedConfig reads seed_config.json from ~/.autocut/config/.
func (s *SeedService) loadSeedConfig() (*seedConfig, error) {
	path := filepath.Join(s.dir.ConfigDir, "seed_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg seedConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// seedChannel creates OAuthClientSecret + Channel + ChannelConfig for one seed entry.
// Skips if a channel with that name already exists.
func (s *SeedService) seedChannel(ctx context.Context, sc seedChannel) error {
	channelRepo := database.NewChannelRepo(s.db, s.log)
	oauthRepo := database.NewOAuthClientSecretRepo(s.db, s.log)
	configRepo := database.NewChannelConfigRepo(s.db, s.log)

	// Idempotency: skip if channel already exists by name.
	existing, err := channelRepo.GetByName(ctx, sc.Name)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("check existing channel: %w", err)
	}
	if existing != nil {
		s.log.Debug("channel already exists, skipping", "channel", sc.Name)
		return nil
	}

	// 1. Import OAuth client secret.
	secretID, err := s.importClientSecret(ctx, oauthRepo, sc)
	if err != nil {
		return fmt.Errorf("import client secret: %w", err)
	}

	// 2. Guard: skip if this OAuth credential is already linked to another channel.
	channels, err := channelRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}
	for _, ch := range channels {
		if ch.OAuthClientSecretID.Valid && ch.OAuthClientSecretID.Int64 == secretID {
			s.log.Debug("OAuth credential already in use, skipping", "channel", sc.Name)
			return nil
		}
	}

	// 3. Create channel.
	channelID, err := channelRepo.Create(ctx, sc.Name)
	if err != nil {
		return fmt.Errorf("create channel: %w", err)
	}

	// Link OAuth secret and apply predefined config (if known YouTube channel ID).
	ch := &database.Channel{
		ID:                  channelID,
		Name:                sc.Name,
		OAuthClientSecretID: sql.NullInt64{Int64: secretID, Valid: true},
	}
	if err := channelRepo.Update(ctx, ch); err != nil {
		return fmt.Errorf("update channel oauth: %w", err)
	}

	// 4. Create channel config with gradient colors from seed + predefined extras.
	if err := configRepo.CreateDefault(ctx, channelID); err != nil {
		return fmt.Errorf("create default config: %w", err)
	}
	cc, err := configRepo.GetByChannelIDOrNil(ctx, channelID)
	if err != nil || cc == nil {
		return fmt.Errorf("get created config: %w", err)
	}

	// Apply gradient from seed_config.json.
	cc.GradientColorStart = sc.GradientColorStart
	cc.GradientColorEnd = sc.GradientColorEnd
	cc.MadeForKids = sc.MadeForKids

	// Apply predefined extras by YouTube channel ID (if available).
	// The channel_id is populated after OAuth flow, so we match by name pattern.
	// We also expose a helper so callers can apply predefined config later once
	// the YouTube channel ID is known.
	applyPredefinedByChannelName(sc.Name, cc)

	if err := configRepo.Update(ctx, cc); err != nil {
		return fmt.Errorf("update channel config: %w", err)
	}

	s.log.Info("seeded channel",
		"channel", sc.Name,
		"gradient", sc.GradientColorStart+"→"+sc.GradientColorEnd,
	)
	return nil
}

// importClientSecret reads a client_secret_*.json from ~/.autocut/config/ and
// creates (or returns existing) OAuthClientSecret in the DB.
func (s *SeedService) importClientSecret(
	ctx context.Context,
	repo *database.OAuthClientSecretRepo,
	sc seedChannel,
) (int64, error) {
	secretName := "OAuth " + sc.Name

	// Return existing if already imported.
	existing, err := repo.GetByName(ctx, secretName)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("check existing secret: %w", err)
	}
	if existing != nil {
		return existing.ID, nil
	}

	// Read client_secret file from config dir.
	path := filepath.Join(s.dir.ConfigDir, sc.ClientSecretFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}

	var csj clientSecretJSON
	if err := json.Unmarshal(data, &csj); err != nil {
		return 0, fmt.Errorf("parse %s: %w", sc.ClientSecretFile, err)
	}

	// First secret inserted becomes the default.
	all, err := repo.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("list secrets: %w", err)
	}
	isDefault := len(all) == 0

	id, err := repo.Create(ctx, &database.OAuthClientSecret{
		Name:         secretName,
		ClientID:     csj.Installed.ClientID,
		ClientSecret: csj.Installed.ClientSecret,
		ProjectID:    csj.Installed.ProjectID,
		IsDefault:    isDefault,
	})
	if err != nil {
		return 0, fmt.Errorf("create secret: %w", err)
	}

	s.log.Info("imported OAuth secret", "name", secretName, "is_default", isDefault)
	return id, nil
}

// applyPredefinedByChannelName applies predefined config values matched
// by a name suffix (e.g., "cortes_inerd" → iNerd YouTube channel).
// The full YouTube channel ID is only known after OAuth flow, so we
// use name matching as a heuristic for the initial seed.
func applyPredefinedByChannelName(name string, cc *database.ChannelConfig) {
	var match *channelDefaults

	// Match name → YouTube channel ID heuristic.
	nameToID := map[string]string{
		"cortes_inerd":   "UCpjSuvMTQAq6SKuRE_SoaDQ",
		"cortes_maromba": "UCpGpj9G3W1ofZGG2kcIWvnQ",
		"cortes_react":   "UCNcaOXuxyjSwrbEhgZWA-NA",
	}

	if ytID, ok := nameToID[name]; ok {
		if d, ok := predefinedChannels[ytID]; ok {
			match = &d
		}
	}

	if match == nil {
		return
	}

	cc.DefaultCategoryID = match.categoryID
	cc.DefaultPlaylistIDCortes = sql.NullString{String: match.playlistIDCortes, Valid: match.playlistIDCortes != ""}
	cc.DefaultPlaylistIDShorts = sql.NullString{String: match.playlistIDShorts, Valid: match.playlistIDShorts != ""}
	cc.DefaultTags = match.tags
	cc.PinnedCommentTemplate = match.pinnedComment

	// Only override gradient colors from predefined if seed_config didn't set them
	// (seed_config colors take precedence — they're already set before this call).
}

// ApplyPredefinedByYouTubeID applies predefined config for a channel once
// its YouTube channel ID is known (i.e., after OAuth flow completes).
// This should be called from the OAuth callback handler.
func ApplyPredefinedByYouTubeID(ytChannelID string, cc *database.ChannelConfig) bool {
	d, ok := predefinedChannels[ytChannelID]
	if !ok {
		return false
	}
	cc.DefaultCategoryID = d.categoryID
	cc.DefaultPlaylistIDCortes = sql.NullString{String: d.playlistIDCortes, Valid: d.playlistIDCortes != ""}
	cc.DefaultPlaylistIDShorts = sql.NullString{String: d.playlistIDShorts, Valid: d.playlistIDShorts != ""}
	cc.DefaultTags = d.tags
	cc.PinnedCommentTemplate = d.pinnedComment
	if d.gradientColorStart != "" {
		cc.GradientColorStart = d.gradientColorStart
	}
	if d.gradientColorEnd != "" {
		cc.GradientColorEnd = d.gradientColorEnd
	}
	return true
}
