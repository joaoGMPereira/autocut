package database

import "database/sql"

type migration struct {
	version int
	name    string
	fn      func(*sql.Tx) error
}

// allMigrations is the ordered list of schema migrations.
// Add new migrations to the end with incrementing version numbers.
var allMigrations = []migration{
	{version: 1, name: "initial_schema", fn: migrateV1},
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
