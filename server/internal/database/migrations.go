package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type migration struct {
	version int
	name    string
	fn      func(*sql.Tx) error
}

// allMigrations is the ordered list of schema migrations.
// Add new migrations to the end with incrementing version numbers.
var allMigrations = []migration{
	{version: 1, name: "initial_schema", fn: migrateV1},
	{version: 2, name: "pipeline_parity", fn: migrateV2},
	{version: 3, name: "mode_config_json", fn: migrateV3},
	{version: 4, name: "music_blacklist", fn: migrateV4},
	{version: 5, name: "queue_columns", fn: migrateV5},
	{version: 6, name: "remote_videos", fn: migrateV6},
	{version: 7, name: "video_comments", fn: migrateV7},
	{version: 8, name: "pipeline_source_run_id", fn: migrateV8},
	{version: 9, name: "pipeline_rewrite", fn: migrateV9},
	{version: 10, name: "pipeline_run_video_meta", fn: migrateV10},
	{version: 11, name: "url_history", fn: migrateV11},
	{version: 12, name: "seed_channels_oauth", fn: migrateV12},
	{version: 13, name: "seed_app_settings_from_channel", fn: migrateV13},
	{version: 14, name: "seed_channel_configs_and_defaults", fn: migrateV14},
	{version: 15, name: "seed_overlay_individual_keys", fn: migrateV15},
	{version: 16, name: "pipeline_run_thumbnail_url", fn: migrateV16},
	{version: 17, name: "pipeline_clip_thumbnail_text", fn: migrateV17},
}

// migrateV1 creates all 11 application tables.
// Ported 1:1 from Kotlin Tables.kt + QuotaRepository.kt (QuotaUsageTable).
func migrateV1(tx *sql.Tx) error {
	_, err := tx.Exec(`
		-- 1. oauth_client_secrets (no FK dependencies)
		-- Kotlin: OAuthClientSecretsTable : IntIdTable("oauth_client_secrets")
		CREATE TABLE oauth_client_secrets (
			id              INTEGER PRIMARY KEY,
			name            TEXT    NOT NULL UNIQUE,
			client_id       TEXT    NOT NULL,
			client_secret   TEXT    NOT NULL,
			project_id      TEXT    NOT NULL DEFAULT '',
			is_default      INTEGER NOT NULL DEFAULT 0,
			created_at      INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),
			updated_at      INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 2. app_settings (no FK dependencies)
		-- Kotlin: AppSettingsTable : IntIdTable("app_settings")
		CREATE TABLE app_settings (
			id          INTEGER PRIMARY KEY,
			key         TEXT    NOT NULL UNIQUE,
			value       TEXT    NOT NULL DEFAULT '',
			updated_at  INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 3. channels (FK → oauth_client_secrets)
		-- Kotlin: ChannelsTable : IntIdTable("channels")
		CREATE TABLE channels (
			id                      INTEGER PRIMARY KEY,
			name                    TEXT    NOT NULL UNIQUE,
			channel_id              TEXT    NOT NULL DEFAULT '',
			channel_title           TEXT    NOT NULL DEFAULT '',
			avatar_url              TEXT    NOT NULL DEFAULT '',
			access_token            TEXT    NOT NULL DEFAULT '',
			refresh_token           TEXT    NOT NULL DEFAULT '',
			expires_at              INTEGER NOT NULL DEFAULT 0,
			oauth_client_secret_id  INTEGER REFERENCES oauth_client_secrets(id) ON DELETE SET NULL,
			is_favorite             INTEGER NOT NULL DEFAULT 0,
			created_at              INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),
			updated_at              INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 4. videos (FK → channels)
		-- Kotlin: VideosTable : IntIdTable("videos")
		CREATE TABLE videos (
			id          INTEGER PRIMARY KEY,
			channel_id  INTEGER REFERENCES channels(id) ON DELETE SET NULL,
			url         TEXT    NOT NULL,
			platform    TEXT    NOT NULL,
			title       TEXT    NOT NULL DEFAULT '',
			file_path   TEXT    NOT NULL,
			duration    INTEGER NOT NULL DEFAULT 0,
			status      TEXT    NOT NULL DEFAULT 'downloaded',
			created_at  INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 5. clips (FK → videos)
		-- Kotlin: ClipsTable : IntIdTable("clips")
		CREATE TABLE clips (
			id              INTEGER PRIMARY KEY,
			video_id        INTEGER NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
			start_time      REAL    NOT NULL,
			end_time        REAL    NOT NULL,
			file_path       TEXT    NOT NULL,
			title           TEXT    NOT NULL DEFAULT '',
			description     TEXT    NOT NULL DEFAULT '',
			tags            TEXT    NOT NULL DEFAULT '',
			thumbnail_path  TEXT    NOT NULL DEFAULT '',
			status          TEXT    NOT NULL DEFAULT 'created',
			created_at      INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 6. uploads (FK → clips, channels)
		-- Kotlin: UploadsTable : IntIdTable("uploads")
		CREATE TABLE uploads (
			id                   INTEGER PRIMARY KEY,
			clip_id              INTEGER NOT NULL REFERENCES clips(id) ON DELETE CASCADE,
			channel_id           INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
			youtube_id           TEXT    NOT NULL DEFAULT '',
			youtube_url          TEXT    NOT NULL DEFAULT '',
			status               TEXT    NOT NULL DEFAULT 'pending',
			scheduled_at         INTEGER,
			uploaded_at          INTEGER,
			error                TEXT    NOT NULL DEFAULT '',
			video_type           TEXT    NOT NULL DEFAULT 'long_form',
			local_video_path     TEXT,
			local_thumbnail_path TEXT,
			metadata_json        TEXT    NOT NULL DEFAULT '{}',
			upload_config_json   TEXT    NOT NULL DEFAULT '{}',
			original_video_name  TEXT    NOT NULL DEFAULT '',
			source_video_url     TEXT    NOT NULL DEFAULT '',
			source_clip_url      TEXT    NOT NULL DEFAULT '',
			shorts_generated     INTEGER NOT NULL DEFAULT 0,
			shorts_generated_at  INTEGER,
			created_at           INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 7. pipeline_runs (FK → channels)
		-- Kotlin: PipelineRunsTable : IntIdTable("pipeline_runs")
		CREATE TABLE pipeline_runs (
			id           INTEGER PRIMARY KEY,
			channel_id   INTEGER REFERENCES channels(id) ON DELETE SET NULL,
			url          TEXT    NOT NULL,
			mode         TEXT    NOT NULL,
			status       TEXT    NOT NULL DEFAULT 'pending',
			progress     INTEGER NOT NULL DEFAULT 0,
			current_step TEXT    NOT NULL DEFAULT '',
			error        TEXT    NOT NULL DEFAULT '',
			started_at   INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),
			finished_at  INTEGER
		);

		-- 8. thumbnail_backgrounds (FK → channels)
		-- Kotlin: ThumbnailBackgroundsTable : IntIdTable("thumbnail_backgrounds")
		CREATE TABLE thumbnail_backgrounds (
			id          INTEGER PRIMARY KEY,
			channel_id  INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
			name        TEXT    NOT NULL,
			file_path   TEXT    NOT NULL,
			is_default  INTEGER NOT NULL DEFAULT 0,
			created_at  INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 9. channel_config (FK → channels, 1:1 relationship)
		-- Kotlin: ChannelConfigTable : IntIdTable("channel_config")
		CREATE TABLE channel_config (
			id                                INTEGER PRIMARY KEY,
			channel_id                        INTEGER NOT NULL UNIQUE REFERENCES channels(id) ON DELETE CASCADE,

			-- Gradient colors (thumbnail generation)
			gradient_color_start              TEXT    NOT NULL DEFAULT '#F27121',
			gradient_color_end                TEXT    NOT NULL DEFAULT '#8A2387',

			-- Thumbnail text styling
			thumbnail_font_family             TEXT    NOT NULL DEFAULT 'Impact',
			thumbnail_text_color_1            TEXT    NOT NULL DEFAULT '#FFFFFF',
			thumbnail_text_color_2            TEXT    NOT NULL DEFAULT '#FFFF00',
			thumbnail_text_color_3            TEXT    NOT NULL DEFAULT '#000000',

			-- YouTube metadata defaults
			made_for_kids                     INTEGER NOT NULL DEFAULT 0,
			default_category_id               INTEGER NOT NULL DEFAULT 22,
			default_playlist_id_cortes        TEXT,
			default_playlist_id_shorts        TEXT,
			default_tags                      TEXT    NOT NULL DEFAULT '',
			shorts_title_hashtags             TEXT    NOT NULL DEFAULT '',
			video_description_hashtags        TEXT    NOT NULL DEFAULT '',

			-- Anti-duplicate configuration
			anti_duplicate_enabled            INTEGER NOT NULL DEFAULT 0,
			anti_duplicate_mode               TEXT    NOT NULL DEFAULT 'SUBTLE',

			-- Speed configuration
			speed_enabled                     INTEGER NOT NULL DEFAULT 1,
			speed_factor                      REAL    NOT NULL DEFAULT 1.02,

			-- Visual configuration
			visual_crop_enabled               INTEGER NOT NULL DEFAULT 1,
			visual_crop_percent               REAL    NOT NULL DEFAULT 0.02,
			visual_zoom_enabled               INTEGER NOT NULL DEFAULT 0,
			visual_zoom_amount                REAL    NOT NULL DEFAULT 1.03,
			visual_color_grading_enabled      INTEGER NOT NULL DEFAULT 1,
			visual_brightness                 REAL    NOT NULL DEFAULT 0.02,
			visual_saturation                 REAL    NOT NULL DEFAULT 1.05,
			visual_contrast                   REAL    NOT NULL DEFAULT 1.02,

			-- Branding configuration
			branding_logo_enabled             INTEGER NOT NULL DEFAULT 0,
			branding_logo_path                TEXT,
			branding_logo_position            TEXT    NOT NULL DEFAULT 'BOTTOM_RIGHT',
			branding_logo_opacity             REAL    NOT NULL DEFAULT 0.3,
			branding_logo_scale               REAL    NOT NULL DEFAULT 0.1,
			branding_intro_enabled            INTEGER NOT NULL DEFAULT 0,
			branding_intro_path               TEXT,
			branding_generate_intro_from_logo INTEGER NOT NULL DEFAULT 1,
			branding_intro_duration           REAL    NOT NULL DEFAULT 3.0,
			branding_outro_enabled            INTEGER NOT NULL DEFAULT 0,
			branding_outro_path               TEXT,

			-- Audio configuration
			audio_music_enabled               INTEGER NOT NULL DEFAULT 0,
			audio_music_path                  TEXT,
			audio_music_volume                REAL    NOT NULL DEFAULT 0.08,
			audio_random_music                INTEGER NOT NULL DEFAULT 1,
			audio_music_folder                TEXT    NOT NULL DEFAULT 'data/channel_assets/shared/music',

			-- Pinned comment
			pinned_comment_template           TEXT    NOT NULL DEFAULT '',

			-- Preview/caption configuration
			preview_captions_enabled          INTEGER NOT NULL DEFAULT 0,
			preview_caption_style             TEXT    NOT NULL DEFAULT 'BOLD',
			preview_caption_text_style_json   TEXT    NOT NULL DEFAULT '{}',
			preview_video_overlay_config_json TEXT    NOT NULL DEFAULT '{}',
			preview_text_overlay_config_json  TEXT    NOT NULL DEFAULT '{}',

			-- Max highlights
			max_highlights                    INTEGER,

			created_at                        INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),
			updated_at                        INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 10. shorts (FK → channels)
		-- Kotlin: ShortsTable : IntIdTable("shorts")
		CREATE TABLE shorts (
			id                    INTEGER PRIMARY KEY,
			source_video_path     TEXT    NOT NULL,
			segment_json          TEXT    NOT NULL,
			file_path             TEXT    NOT NULL,
			thumbnail_path        TEXT,
			custom_thumbnail_path TEXT,
			title                 TEXT    NOT NULL DEFAULT '',
			thumbnail_title       TEXT,
			description           TEXT    NOT NULL DEFAULT '',
			tags                  TEXT    NOT NULL DEFAULT '',
			duration              REAL    NOT NULL DEFAULT 0.0,
			channel_id            INTEGER REFERENCES channels(id) ON DELETE SET NULL,
			youtube_id            TEXT,
			youtube_url           TEXT,
			status                TEXT    NOT NULL DEFAULT 'created',
			scheduled_at          INTEGER,
			uploaded_at           INTEGER,
			error                 TEXT    NOT NULL DEFAULT '',
			created_at            INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);

		-- 11. quota_usage (FK → oauth_client_secrets)
		-- Kotlin: QuotaUsageTable in QuotaRepository.kt
		CREATE TABLE quota_usage (
			id                      INTEGER PRIMARY KEY,
			oauth_client_secret_id  INTEGER NOT NULL REFERENCES oauth_client_secrets(id) ON DELETE CASCADE,
			date                    TEXT    NOT NULL,
			units_used              INTEGER NOT NULL DEFAULT 0,
			upload_count            INTEGER NOT NULL DEFAULT 0,
			thumbnail_count         INTEGER NOT NULL DEFAULT 0,
			other_api_calls         INTEGER NOT NULL DEFAULT 0,
			created_at              INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),
			updated_at              INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),
			UNIQUE(oauth_client_secret_id, date)
		);

		-- Indexes on FK columns and frequently queried columns
		CREATE INDEX idx_channels_oauth_secret    ON channels(oauth_client_secret_id);
		CREATE INDEX idx_videos_channel_id        ON videos(channel_id);
		CREATE INDEX idx_videos_status            ON videos(status);
		CREATE INDEX idx_clips_video_id           ON clips(video_id);
		CREATE INDEX idx_clips_status             ON clips(status);
		CREATE INDEX idx_uploads_clip_id          ON uploads(clip_id);
		CREATE INDEX idx_uploads_channel_id       ON uploads(channel_id);
		CREATE INDEX idx_uploads_status           ON uploads(status);
		CREATE INDEX idx_pipeline_runs_channel_id ON pipeline_runs(channel_id);
		CREATE INDEX idx_pipeline_runs_status     ON pipeline_runs(status);
		CREATE INDEX idx_thumb_bg_channel_id      ON thumbnail_backgrounds(channel_id);
		CREATE INDEX idx_shorts_channel_id        ON shorts(channel_id);
		CREATE INDEX idx_shorts_status            ON shorts(status);
		CREATE INDEX idx_quota_usage_date         ON quota_usage(date);
	`)
	return err
}

// migrateV3 adds mode_config_json to pipeline_runs for structured mode configuration storage.
func migrateV3(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE pipeline_runs ADD COLUMN mode_config_json TEXT NOT NULL DEFAULT '{}';`)
	return err
}

// migrateV4 creates the music_blacklist table for per-channel music pattern exclusion.
// FP-009: music blacklist management.
func migrateV4(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS music_blacklist (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id INTEGER NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
			pattern    TEXT    NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_music_blacklist_channel_id ON music_blacklist(channel_id);
	`)
	return err
}

// migrateV5 adds queue_order and publish_at columns to uploads for FP-012/FP-013.
// queue_order: integer position in the upload queue (default 0).
// publish_at: ISO8601 scheduled publish time (nullable TEXT).
func migrateV5(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE uploads ADD COLUMN queue_order INTEGER NOT NULL DEFAULT 0;
		ALTER TABLE uploads ADD COLUMN publish_at  TEXT;
	`)
	return err
}

// migrateV2 adds step_outputs_json to pipeline_runs and creates pipeline_run_steps.
func migrateV2(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE pipeline_runs ADD COLUMN step_outputs_json TEXT NOT NULL DEFAULT '{}';

		CREATE TABLE pipeline_run_steps (
			id          INTEGER PRIMARY KEY,
			run_id      INTEGER NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
			step        TEXT    NOT NULL,
			status      TEXT    NOT NULL DEFAULT 'pending',
			job_id      TEXT    NOT NULL DEFAULT '',
			error       TEXT    NOT NULL DEFAULT '',
			started_at  INTEGER,
			finished_at INTEGER
		);

		CREATE INDEX idx_pipeline_run_steps_run_id ON pipeline_run_steps(run_id);
	`)
	return err
}

// migrateV6 creates the remote_videos table for FP-015 (Mass Update).
// Caches YouTube video metadata fetched via API; 1h TTL enforced at query layer.
func migrateV6(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS remote_videos (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id   INTEGER NOT NULL,
			youtube_id   TEXT    NOT NULL,
			title        TEXT,
			description  TEXT,
			tags         TEXT,
			category_id  INTEGER,
			published_at DATETIME,
			fetched_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(channel_id, youtube_id)
		);
		CREATE INDEX IF NOT EXISTS idx_remote_videos_channel  ON remote_videos(channel_id);
		CREATE INDEX IF NOT EXISTS idx_remote_videos_fetched  ON remote_videos(fetched_at);
		CREATE INDEX IF NOT EXISTS idx_remote_videos_pub      ON remote_videos(published_at);
	`)
	return err
}

// migrateV8 adds source_run_id to pipeline_runs to support the "Refazer" feature.
// When source_run_id is set, the pipeline reuses the video file from the original run
// instead of downloading it again.
func migrateV8(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE pipeline_runs ADD COLUMN source_run_id INTEGER REFERENCES pipeline_runs(id);`)
	return err
}

// migrateV9 replaces the old pipeline_runs / pipeline_run_steps tables with
// the new state-machine schema (state, active_phase, highlights, clips).
func migrateV9(tx *sql.Tx) error {
	_, err := tx.Exec(`
		-- Drop old tables (cascades handled by FK ON DELETE)
		DROP TABLE IF EXISTS pipeline_run_steps;
		DROP TABLE IF EXISTS pipeline_runs;

		-- New pipeline_runs: state-machine driven
		CREATE TABLE pipeline_runs (
			id               INTEGER PRIMARY KEY,
			channel_id       INTEGER REFERENCES channels(id) ON DELETE SET NULL,
			url              TEXT    NOT NULL DEFAULT '',
			mode             TEXT    NOT NULL DEFAULT 'ai',
			state            TEXT    NOT NULL DEFAULT 'WAITING_URL',
			active_phase     TEXT    NOT NULL DEFAULT '',
			error            TEXT    NOT NULL DEFAULT '',
			mode_config_json TEXT    NOT NULL DEFAULT '{}',
			video_path       TEXT    NOT NULL DEFAULT '',
			transcript_path  TEXT    NOT NULL DEFAULT '',
			started_at       INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000),
			finished_at      INTEGER
		);
		CREATE INDEX idx_pipeline_runs_channel ON pipeline_runs(channel_id);
		CREATE INDEX idx_pipeline_runs_state   ON pipeline_runs(state);

		-- AI highlight review data
		CREATE TABLE pipeline_highlights (
			id           INTEGER PRIMARY KEY,
			run_id       INTEGER NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
			start_sec    REAL    NOT NULL DEFAULT 0,
			end_sec      REAL    NOT NULL DEFAULT 0,
			adj_start_sec REAL   NOT NULL DEFAULT 0,
			adj_end_sec  REAL    NOT NULL DEFAULT 0,
			score        REAL    NOT NULL DEFAULT 0,
			text         TEXT    NOT NULL DEFAULT '',
			reason       TEXT    NOT NULL DEFAULT '',
			is_selected  INTEGER NOT NULL DEFAULT 1,
			created_at   INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);
		CREATE INDEX idx_pipeline_highlights_run ON pipeline_highlights(run_id);

		-- Generated clips with metadata and upload tracking
		CREATE TABLE pipeline_clips (
			id              INTEGER PRIMARY KEY,
			run_id          INTEGER NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
			highlight_id    INTEGER REFERENCES pipeline_highlights(id) ON DELETE SET NULL,
			file_path       TEXT    NOT NULL DEFAULT '',
			thumbnail_path  TEXT    NOT NULL DEFAULT '',
			title           TEXT    NOT NULL DEFAULT '',
			description     TEXT    NOT NULL DEFAULT '',
			tags            TEXT    NOT NULL DEFAULT '',
			thumbnail_style TEXT    NOT NULL DEFAULT 'default',
			is_selected     INTEGER NOT NULL DEFAULT 1,
			start_sec       REAL    NOT NULL DEFAULT 0,
			end_sec         REAL    NOT NULL DEFAULT 0,
			duration_sec    REAL    NOT NULL DEFAULT 0,
			upload_status   TEXT    NOT NULL DEFAULT 'pending',
			youtube_id      TEXT    NOT NULL DEFAULT '',
			created_at      INTEGER NOT NULL DEFAULT (CAST(strftime('%s','now') AS INTEGER) * 1000)
		);
		CREATE INDEX idx_pipeline_clips_run       ON pipeline_clips(run_id);
		CREATE INDEX idx_pipeline_clips_highlight ON pipeline_clips(highlight_id);
	`)
	return err
}

// migrateV10 adds video_title and duration_sec to pipeline_runs.
// video_title: human-readable title from yt-dlp metadata.
// duration_sec: video duration in whole seconds, required by downstream phases.
func migrateV10(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE pipeline_runs ADD COLUMN video_title  TEXT    NOT NULL DEFAULT '';
		ALTER TABLE pipeline_runs ADD COLUMN duration_sec INTEGER NOT NULL DEFAULT 0;
	`)
	return err
}

// migrateV11 creates the url_history table for URL autocomplete/history (099).
func migrateV11(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS url_history (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			url          TEXT    NOT NULL UNIQUE,
			video_title  TEXT,
			last_used_at INTEGER NOT NULL,
			use_count    INTEGER NOT NULL DEFAULT 1
		)
	`)
	return err
}

// migrateV12 seeds the 4 known OAuth client secret profiles and 4 channels extracted from the
// Youtube companion app's database. Uses INSERT OR IGNORE so re-running on an existing database
// (e.g. after a future re-install) is a no-op — existing records are never overwritten.
// Channels are seeded without tokens (access_token/refresh_token empty) so each one starts
// as "Not Authorized" and the user completes the OAuth flow once per channel after install.
func migrateV12(tx *sql.Tx) error {
	_, err := tx.Exec(`
		-- OAuth client secret profiles (app-level credentials, safe to embed)
		INSERT OR IGNORE INTO oauth_client_secrets (name, client_id, client_secret, project_id, is_default) VALUES
			('OAuth cortes_inerd',          'PLACEHOLDER_CLIENT_ID_INERD', 'PLACEHOLDER_CLIENT_SECRET_INERD', 'video-clipper-484223',        1),
			('OAuth cortes_maromba',        'PLACEHOLDER_CLIENT_ID_MAROMBA','PLACEHOLDER_CLIENT_SECRET_MAROMBA', 'video-clipper-484209',        0),
			('OAuth cortes_react',          'PLACEHOLDER_CLIENT_ID_REACT', 'PLACEHOLDER_CLIENT_SECRET_REACT', 'video-clipper-484017',        0),
			('gen-lang-client-0861870513',  'PLACEHOLDER_CLIENT_ID_GEN_LANG', 'PLACEHOLDER_CLIENT_SECRET_GEN_LANG', 'gen-lang-client-0861870513', 0);

		-- Channels (no tokens — user must authorize via OAuth flow after install)
		INSERT OR IGNORE INTO channels (name, channel_id, channel_title, oauth_client_secret_id, is_favorite) VALUES
			('dev_ao_cubo',         'UCSwTDK61hsBFV4CToQtFl-g', 'Dev ao Cubo',                    (SELECT id FROM oauth_client_secrets WHERE name = 'gen-lang-client-0861870513'), 1),
			('cortes_maromba',      'UCpGpj9G3W1ofZGG2kcIWvnQ', 'Cortes da Maromba',              (SELECT id FROM oauth_client_secrets WHERE name = 'OAuth cortes_maromba'),        0),
			('cortes_inerd',        'UCpjSuvMTQAq6SKuRE_SoaDQ', 'Cortes do Velho Peter (Ei Nerd)',(SELECT id FROM oauth_client_secrets WHERE name = 'OAuth cortes_inerd'),          0),
			('cortes_react',        'UCNcaOXuxyjSwrbEhgZWA-NA', 'Cortes e React',                 (SELECT id FROM oauth_client_secrets WHERE name = 'OAuth cortes_react'),          0);
	`)
	return err
}

// migrateV13 seeds ~35 app_settings keys from the favorite channel's config on fresh install.
// If no channels exist, only the fixed defaults are seeded.
// Uses INSERT OR IGNORE so existing settings are never overwritten.
func migrateV13(tx *sql.Tx) error {
	const insertSQL = `INSERT OR IGNORE INTO app_settings (key, value, updated_at) VALUES (?, ?, CAST(strftime('%s','now') AS INTEGER) * 1000)`

	// Helper to seed a key-value pair.
	seed := func(key, value string) error {
		_, err := tx.Exec(insertSQL, key, value)
		if err != nil {
			return fmt.Errorf("seed app_setting %q: %w", key, err)
		}
		return nil
	}

	// Helper to format a bool as "true"/"false".
	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	// --- Find the favorite channel (or first channel) ---
	var channelID int64
	var hasChannel bool

	err := tx.QueryRow(`SELECT id FROM channels WHERE is_favorite = 1 ORDER BY id LIMIT 1`).Scan(&channelID)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(`SELECT id FROM channels ORDER BY id LIMIT 1`).Scan(&channelID)
	}
	if err == sql.ErrNoRows {
		hasChannel = false
	} else if err != nil {
		return fmt.Errorf("migrateV13: find channel: %w", err)
	} else {
		hasChannel = true
	}

	// --- Seed from channel_config if a channel exists ---
	if hasChannel {
		var (
			brandingLogoEnabled  bool
			brandingLogoPath     sql.NullString
			brandingLogoPosition string
			brandingLogoOpacity  float64
			brandingLogoScale    float64
			brandingIntroEnabled bool
			brandingIntroPath    sql.NullString
			brandingOutroEnabled bool
			brandingOutroPath    sql.NullString
			captionsEnabled      bool
			captionStyle         string
			overlayConfigJSON    string
			musicEnabled         bool
			musicPath            sql.NullString
			musicVolume          float64
			antiDupEnabled       bool
			antiDupMode          string
		)

		err = tx.QueryRow(`SELECT
			branding_logo_enabled, branding_logo_path, branding_logo_position, branding_logo_opacity, branding_logo_scale,
			branding_intro_enabled, branding_intro_path,
			branding_outro_enabled, branding_outro_path,
			preview_captions_enabled, preview_caption_style,
			preview_video_overlay_config_json,
			audio_music_enabled, audio_music_path, audio_music_volume,
			anti_duplicate_enabled, anti_duplicate_mode
			FROM channel_config WHERE channel_id = ?`, channelID).Scan(
			&brandingLogoEnabled, &brandingLogoPath, &brandingLogoPosition, &brandingLogoOpacity, &brandingLogoScale,
			&brandingIntroEnabled, &brandingIntroPath,
			&brandingOutroEnabled, &brandingOutroPath,
			&captionsEnabled, &captionStyle,
			&overlayConfigJSON,
			&musicEnabled, &musicPath, &musicVolume,
			&antiDupEnabled, &antiDupMode,
		)

		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("migrateV13: read channel_config: %w", err)
		}

		if err == nil {
			// We have channel_config — map fields to app_settings.

			// branding_logo_enabled: force false if path is NULL or empty
			logoEnabled := brandingLogoEnabled && brandingLogoPath.Valid && brandingLogoPath.String != ""
			if err := seed("branding_logo_enabled", boolStr(logoEnabled)); err != nil {
				return err
			}
			// branding_logo_path: only insert if non-NULL/non-empty
			if brandingLogoPath.Valid && brandingLogoPath.String != "" {
				if err := seed("branding_logo_path", brandingLogoPath.String); err != nil {
					return err
				}
			}
			if err := seed("branding_logo_position", brandingLogoPosition); err != nil {
				return err
			}
			if err := seed("branding_logo_opacity", fmt.Sprintf("%d", int(brandingLogoOpacity*100))); err != nil {
				return err
			}
			if err := seed("branding_logo_scale", fmt.Sprintf("%d", int(brandingLogoScale*100))); err != nil {
				return err
			}

			// branding_intro_enabled: force false if path is NULL
			introEnabled := brandingIntroEnabled && brandingIntroPath.Valid
			if err := seed("branding_intro_enabled", boolStr(introEnabled)); err != nil {
				return err
			}

			// branding_outro_enabled: force false if path is NULL
			outroEnabled := brandingOutroEnabled && brandingOutroPath.Valid
			if err := seed("branding_outro_enabled", boolStr(outroEnabled)); err != nil {
				return err
			}

			if err := seed("captions_enabled", boolStr(captionsEnabled)); err != nil {
				return err
			}
			if err := seed("captions_preset", captionStyle); err != nil {
				return err
			}
			if err := seed("overlay_config_json", overlayConfigJSON); err != nil {
				return err
			}

			if err := seed("music_enabled", boolStr(musicEnabled)); err != nil {
				return err
			}
			// music_custom_path: only if non-NULL/non-empty
			if musicPath.Valid && musicPath.String != "" {
				if err := seed("music_custom_path", musicPath.String); err != nil {
					return err
				}
			}
			if err := seed("music_volume_pct", fmt.Sprintf("%d", int(musicVolume*100))); err != nil {
				return err
			}

			if err := seed("anti_dup_enabled", boolStr(antiDupEnabled)); err != nil {
				return err
			}
			if err := seed("anti_dup_mode", antiDupMode); err != nil {
				return err
			}
		}
		// If err == sql.ErrNoRows (no config), we skip channel-derived settings
		// and just seed the fixed defaults below.
	}

	// --- Fixed defaults (always seed) ---
	fixedDefaults := [][2]string{
		{"workflow_mode_default", "ai"},
		{"ai_sensitivity_pct", "70"},
		{"ai_clip_duration_secs", "1200"},
		{"ai_min_duration_secs", "480"},
		{"ai_force_regenerate", "true"},
		{"ai_skip_transcription", "false"},
		{"upload_privacy", "private"},
		{"upload_dry_run", "false"},
		{"anti_dup_effect_speed_boost", "true"},
		{"anti_dup_effect_crop", "true"},
		{"anti_dup_effect_color_grading", "true"},
		{"anti_dup_effect_noise", "true"},
		{"anti_dup_effect_noise_strength", "3"},
		{"anti_dup_effect_zoom", "true"},
		{"anti_dup_effect_transitions", "true"},
	}
	for _, kv := range fixedDefaults {
		if err := seed(kv[0], kv[1]); err != nil {
			return err
		}
	}

	return nil
}

// migrateV14 seeds channel_config rows for all 4 predefined channels with real values
// from the Kotlin Youtube companion app (PredefinedChannels.kt + channel_preferences.json).
// Also re-seeds app_settings with correct enabled defaults since v13 got NULLs.
// Uses INSERT OR IGNORE for channel_config so existing configs are never overwritten.
func migrateV14(tx *sql.Tx) error {
	now := fmt.Sprintf("%d", nowMillis())

	// Per-channel config data from PredefinedChannels.kt
	type channelSeed struct {
		youtubeID          string
		gradientStart      string
		gradientEnd        string
		categoryID         int
		playlistCortes     string
		playlistShorts     string
		tags               string
		pinnedComment      string
	}

	channels := []channelSeed{
		{
			youtubeID:      "UCSwTDK61hsBFV4CToQtFl-g",
			gradientStart:  "#FAF98C",
			gradientEnd:    "#176BFF",
			categoryID:     28,
			playlistCortes: "PLz4FQFPuKNaz7blXwx_pgNBdk8KWEzWwC",
			playlistShorts: "PLz4FQFPuKNazfULbLOaiwPykDBBhzRrZW",
			tags:           "javascript,canvas,gemini,html,programação,css,vibecoding,gameplay,gamedev,jogo no navegador,ia,ia vs ia,codando com ia,criando jogos com ia,google gemini,inteligência artificial,chatgpt,desenvolvimento de jogos,jogos com ia,do zero",
			pinnedComment:  "Se inscreva no canal, comente e deixe o like! ❤️",
		},
		{
			youtubeID:      "UCpGpj9G3W1ofZGG2kcIWvnQ",
			gradientStart:  "#C80E0E",
			gradientEnd:    "#0F0F0F",
			categoryID:     17,
			playlistCortes: "PL1gfq1_qdCBAOCCaQpuLtO-Byq65B6up8",
			playlistShorts: "PL1gfq1_qdCBCdTiDk3YxVC8WKxXe187ZM",
			tags:           "renato cariani,bodybuilding,bodybuilder,fisiologia,farmacia,farmacologia,treino,musculação,educação fisica,quimica,cariane,emagrecer,dieta,alimentação,saudavel,natural,suplemento,perder peso,receita,fit,toguro,mansão maromba,sabor,sabor energético",
			pinnedComment:  "Se inscreva no canal, comente e deixe o like! ❤️",
		},
		{
			youtubeID:      "UCpjSuvMTQAq6SKuRE_SoaDQ",
			gradientStart:  "#FAE70D",
			gradientEnd:    "#F43414",
			categoryID:     24,
			playlistCortes: "PLKj6HM7XxhZnZhjwjuhTTZsuMdxvciYLT",
			playlistShorts: "PLKj6HM7XxhZlwC-6lHg084F39b8RuaYDJ",
			tags:           "nerd,ei nerd,peter jordan,ucm,marvel,dc,dc comics,cinema,filmes,animes,quadrinhos,hqs,series,geek,cultura pop,seriados,heróis,super herois,peter jordan,peteraqui,einerdtv,ei nerd,cortes peter jordan,cortes einerdtv,cortes ei nerd,canal do peter jordan,canal ei nerd,reação do peter jordan,peter jordan reagindo,react peter jordan,react ei nerd,quadro ei nerd,nerdices do peter jordan,opinião do peter jordan,análise do peter jordan,crítica do peter jordan",
			pinnedComment:  "Se inscreva no canal, comente e deixe o like! ❤️",
		},
		{
			youtubeID:      "UCNcaOXuxyjSwrbEhgZWA-NA",
			gradientStart:  "#F27121",
			gradientEnd:    "#8A2387",
			categoryID:     24,
			playlistCortes: "PLnpzTyt6JdHJ6vSWuiIvv5BXxcdKyi8QY",
			playlistShorts: "PLnpzTyt6JdHKUc4qiASAug0VkSGgFOB2U",
			tags:           "brino,react,engraçado,brunozor,bruninzor,reagindo,assistindo,bruninho,brinozor,videos engraçados,video react,experimentando,comida,viagem,humor,curiosidade,tiktoks,história,entretenimento,luxo",
			pinnedComment:  "Se inscreva no canal, comente e deixe o like! ❤️",
		},
	}

	// Shared processing config from channel_preferences.json
	const insertChannelConfig = `INSERT OR IGNORE INTO channel_config (
		channel_id,
		gradient_color_start, gradient_color_end,
		thumbnail_font_family, thumbnail_text_color_1, thumbnail_text_color_2, thumbnail_text_color_3,
		made_for_kids, default_category_id,
		default_playlist_id_cortes, default_playlist_id_shorts,
		default_tags, pinned_comment_template,
		anti_duplicate_enabled, anti_duplicate_mode,
		speed_enabled, speed_factor,
		visual_crop_enabled, visual_crop_percent,
		visual_zoom_enabled, visual_zoom_amount,
		visual_color_grading_enabled, visual_brightness, visual_saturation, visual_contrast,
		branding_logo_enabled, branding_logo_position, branding_logo_opacity, branding_logo_scale,
		branding_intro_enabled, branding_generate_intro_from_logo, branding_intro_duration,
		branding_outro_enabled,
		audio_music_enabled, audio_music_volume, audio_random_music,
		preview_captions_enabled, preview_caption_style,
		created_at, updated_at
	) VALUES (
		?,
		?, ?,
		'Impact', '#FFFFFF', '#FFFF00', '#000000',
		0, ?,
		?, ?,
		?, ?,
		1, 'SUBTLE',
		1, 1.02,
		1, 0.02,
		0, 1.03,
		1, 0.02, 1.05, 1.02,
		0, 'BOTTOM_RIGHT', 0.3, 0.1,
		0, 1, 3.0,
		0,
		1, 0.08, 1,
		1, 'BOLD',
		?, ?
	)`

	for _, ch := range channels {
		var internalID int64
		err := tx.QueryRow(`SELECT id FROM channels WHERE channel_id = ?`, ch.youtubeID).Scan(&internalID)
		if err == sql.ErrNoRows {
			continue // channel not seeded, skip
		}
		if err != nil {
			return fmt.Errorf("migrateV14: lookup channel %s: %w", ch.youtubeID, err)
		}

		_, err = tx.Exec(insertChannelConfig,
			internalID,
			ch.gradientStart, ch.gradientEnd,
			ch.categoryID,
			ch.playlistCortes, ch.playlistShorts,
			ch.tags, ch.pinnedComment,
			now, now,
		)
		if err != nil {
			return fmt.Errorf("migrateV14: seed channel_config for %s: %w", ch.youtubeID, err)
		}
	}

	// Re-seed app_settings with correct enabled defaults.
	// v13 may have seeded these with empty/false because channel_config didn't exist yet.
	// Only update settings still at their empty/default state.
	type settingUpdate struct {
		key   string
		value string
	}
	updates := []settingUpdate{
		{"anti_dup_enabled", "true"},
		{"anti_dup_mode", "SUBTLE"},
		{"anti_dup_effect_speed_boost", "true"},
		{"anti_dup_effect_crop", "true"},
		{"anti_dup_effect_color_grading", "true"},
		{"anti_dup_effect_noise", "true"},
		{"anti_dup_effect_noise_strength", "3"},
		{"anti_dup_effect_zoom", "true"},
		{"anti_dup_effect_transitions", "true"},
		{"captions_enabled", "true"},
		{"captions_preset", "BOLD"},
		{"music_enabled", "true"},
		{"music_volume_pct", "8"},
		{"branding_logo_position", "BOTTOM_RIGHT"},
		{"branding_logo_opacity", "30"},
		{"branding_logo_scale", "10"},
	}

	insertOrIgnore := `INSERT OR IGNORE INTO app_settings (key, value, updated_at) VALUES (?, ?, ` + now + `)`
	updateIfDefault := `UPDATE app_settings SET value = ?, updated_at = ` + now + ` WHERE key = ? AND (value = '' OR value = 'false' OR value = '0')`

	for _, u := range updates {
		// First try to insert (for fresh installs where v13 didn't seed this key)
		if _, err := tx.Exec(insertOrIgnore, u.key, u.value); err != nil {
			return fmt.Errorf("migrateV14: insert app_setting %q: %w", u.key, err)
		}
		// Then update if the value is still at its empty/default state
		if _, err := tx.Exec(updateIfDefault, u.value, u.key); err != nil {
			return fmt.Errorf("migrateV14: update app_setting %q: %w", u.key, err)
		}
	}

	return nil
}

// migrateV15 seeds individual overlay keys from the channel_config's
// preview_video_overlay_config_json field. The frontend (StepMode.tsx) reads
// 6 individual keys (overlay_enabled, overlay_video_path, etc.) but v13 stored
// a single overlay_config_json blob which the frontend ignores.
func migrateV15(tx *sql.Tx) error {
	const insertSQL = `INSERT OR IGNORE INTO app_settings (key, value, updated_at) VALUES (?, ?, CAST(strftime('%s','now') AS INTEGER) * 1000)`

	seed := func(key, value string) error {
		_, err := tx.Exec(insertSQL, key, value)
		if err != nil {
			return fmt.Errorf("migrateV15: seed app_setting %q: %w", key, err)
		}
		return nil
	}

	// --- Find the favorite channel (or first channel) ---
	var channelID int64
	var hasChannel bool

	err := tx.QueryRow(`SELECT id FROM channels WHERE is_favorite = 1 ORDER BY id LIMIT 1`).Scan(&channelID)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(`SELECT id FROM channels ORDER BY id LIMIT 1`).Scan(&channelID)
	}
	if err == sql.ErrNoRows {
		hasChannel = false
	} else if err != nil {
		return fmt.Errorf("migrateV15: find channel: %w", err)
	} else {
		hasChannel = true
	}

	if hasChannel {
		var overlayJSON string
		err = tx.QueryRow(`SELECT preview_video_overlay_config_json FROM channel_config WHERE channel_id = ?`, channelID).Scan(&overlayJSON)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("migrateV15: read channel_config: %w", err)
		}

		if err == nil && overlayJSON != "" && overlayJSON != "{}" {
			type appearance struct {
				StartPct float64 `json:"start_pct"`
				EndPct   float64 `json:"end_pct"`
			}
			var cfg struct {
				Enabled     bool         `json:"enabled"`
				Path        string       `json:"path"`
				ScalePct    int          `json:"scale_pct"`
				Position    string       `json:"position"`
				ChromaKey   string       `json:"chroma_key"`
				Appearances []appearance `json:"appearances"`
			}

			if jsonErr := json.Unmarshal([]byte(overlayJSON), &cfg); jsonErr == nil {
				// overlay_enabled: force false if path is empty
				enabled := cfg.Enabled && cfg.Path != ""
				if enabled {
					if err := seed("overlay_enabled", "true"); err != nil {
						return err
					}
				} else {
					if err := seed("overlay_enabled", "false"); err != nil {
						return err
					}
				}

				if cfg.Path != "" {
					if err := seed("overlay_video_path", cfg.Path); err != nil {
						return err
					}
				}

				scalePct := cfg.ScalePct
				if scalePct == 0 {
					scalePct = 100
				}
				if err := seed("overlay_scale_pct", fmt.Sprintf("%d", scalePct)); err != nil {
					return err
				}

				position := cfg.Position
				if position == "" {
					position = "mid_center"
				}
				if err := seed("overlay_position", position); err != nil {
					return err
				}

				appearances := len(cfg.Appearances)
				if appearances == 0 {
					appearances = 1
				}
				if err := seed("overlay_appearances", fmt.Sprintf("%d", appearances)); err != nil {
					return err
				}

				chromaKey := cfg.ChromaKey
				if chromaKey == "" {
					chromaKey = "none"
				}
				if err := seed("overlay_chroma_key", chromaKey); err != nil {
					return err
				}
			}
			// If JSON is malformed, skip silently (same pattern as v13).
		}
	}

	// Delete the stale overlay_config_json key seeded by v13.
	if _, err := tx.Exec(`DELETE FROM app_settings WHERE key = 'overlay_config_json'`); err != nil {
		return fmt.Errorf("migrateV15: delete stale overlay_config_json: %w", err)
	}

	return nil
}

// migrateV16 adds thumbnail_url to pipeline_runs for persisting the YouTube thumbnail URL
// across sessions. Previously this was only available via the ephemeral SSE video_info event.
func migrateV16(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE pipeline_runs ADD COLUMN thumbnail_url TEXT NOT NULL DEFAULT '';`)
	return err
}

// migrateV17 adds thumbnail_text to pipeline_clips for AI-generated thumbnail overlay text.
func migrateV17(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE pipeline_clips ADD COLUMN thumbnail_text TEXT NOT NULL DEFAULT '';`)
	return err
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

// migrateV7 creates the video_comments table for FP-016 (Comment Sync).
// Persists top-level YouTube comment threads per video; comment_id is globally unique.
func migrateV7(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS video_comments (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			youtube_video_id TEXT    NOT NULL,
			channel_id       INTEGER NOT NULL,
			comment_id       TEXT    NOT NULL UNIQUE,
			author           TEXT,
			text             TEXT,
			likes            INTEGER DEFAULT 0,
			published_at     DATETIME,
			synced_at        DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_video_comments_channel ON video_comments(channel_id);
		CREATE INDEX IF NOT EXISTS idx_video_comments_video   ON video_comments(youtube_video_id);
	`)
	return err
}
