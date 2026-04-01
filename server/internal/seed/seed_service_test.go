package seed

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/configurator"
	"github.com/joaoGMPereira/autocut/server/internal/database"
	_ "modernc.org/sqlite"
)

// discardLogger returns a slog.Logger that silently discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func setupTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(dir, discardLogger())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, dir
}

func setupTestDir(t *testing.T, root string) *configurator.AutoCutDir {
	t.Helper()
	dir := configurator.NewAutoCutDirFromRoot(root)
	if err := dir.Ensure(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}
	return dir
}

func TestSeedService_Run_CreatesChannels(t *testing.T) {
	db, root := setupTestDB(t)
	dir := setupTestDir(t, root)
	svc := New(db, dir, discardLogger())

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	channelRepo := database.NewChannelRepo(db, discardLogger())
	channels, err := channelRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list channels: %v", err)
	}

	if len(channels) != 3 {
		t.Errorf("expected 3 channels, got %d", len(channels))
	}

	names := make(map[string]bool)
	for _, ch := range channels {
		names[ch.Name] = true
	}
	for _, want := range []string{"cortes_inerd", "cortes_maromba", "cortes_react"} {
		if !names[want] {
			t.Errorf("missing channel %q", want)
		}
	}
}

func TestSeedService_Run_CreatesChannelConfigs(t *testing.T) {
	db, root := setupTestDB(t)
	dir := setupTestDir(t, root)
	svc := New(db, dir, discardLogger())

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	channelRepo := database.NewChannelRepo(db, discardLogger())
	configRepo := database.NewChannelConfigRepo(db, discardLogger())

	ch, err := channelRepo.GetByName(context.Background(), "cortes_inerd")
	if err != nil {
		t.Fatalf("get cortes_inerd: %v", err)
	}

	cc, err := configRepo.GetByChannelIDOrNil(context.Background(), ch.ID)
	if err != nil || cc == nil {
		t.Fatalf("get channel config: %v", err)
	}

	if cc.GradientColorStart != "#FAE70D" {
		t.Errorf("GradientColorStart: got %q, want #FAE70D", cc.GradientColorStart)
	}
	if cc.GradientColorEnd != "#F43414" {
		t.Errorf("GradientColorEnd: got %q, want #F43414", cc.GradientColorEnd)
	}
	if cc.DefaultCategoryID != 24 {
		t.Errorf("DefaultCategoryID: got %d, want 24", cc.DefaultCategoryID)
	}
	if cc.PinnedCommentTemplate == "" {
		t.Error("PinnedCommentTemplate should not be empty")
	}
}

func TestSeedService_Run_Idempotent(t *testing.T) {
	db, root := setupTestDB(t)
	dir := setupTestDir(t, root)
	svc := New(db, dir, discardLogger())

	// Run twice.
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	channelRepo := database.NewChannelRepo(db, discardLogger())
	channels, err := channelRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list channels: %v", err)
	}
	if len(channels) != 3 {
		t.Errorf("expected 3 channels after 2 runs, got %d (duplicates created)", len(channels))
	}
}

func TestSeedService_Run_ExtractsMusicFiles(t *testing.T) {
	db, root := setupTestDB(t)
	dir := setupTestDir(t, root)
	svc := New(db, dir, discardLogger())

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	entries, err := os.ReadDir(dir.SharedMusicDir)
	if err != nil {
		t.Fatalf("read music dir: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 music files, got %d", len(entries))
	}
}

func TestSeedService_Run_ExtractsSeedConfig(t *testing.T) {
	db, root := setupTestDB(t)
	dir := setupTestDir(t, root)
	svc := New(db, dir, discardLogger())

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir.ConfigDir, "seed_config.json")); err != nil {
		t.Errorf("seed_config.json not extracted: %v", err)
	}
	for _, name := range []string{"client_secret_inerd.json", "client_secret_maromba.json", "client_secret_react.json"} {
		if _, err := os.Stat(filepath.Join(dir.ConfigDir, name)); err != nil {
			t.Errorf("%s not extracted: %v", name, err)
		}
	}
}

func TestSeedService_Run_CreatesOAuthSecrets(t *testing.T) {
	db, root := setupTestDB(t)
	dir := setupTestDir(t, root)
	svc := New(db, dir, discardLogger())

	if err := svc.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	oauthRepo := database.NewOAuthClientSecretRepo(db, discardLogger())
	secrets, err := oauthRepo.List(context.Background())
	if err != nil {
		t.Fatalf("list secrets: %v", err)
	}
	if len(secrets) != 3 {
		t.Errorf("expected 3 OAuth secrets, got %d", len(secrets))
	}

	// First secret should be default.
	var hasDefault bool
	for _, s := range secrets {
		if s.IsDefault {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		t.Error("expected one secret to be default")
	}
}
