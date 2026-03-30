package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/joaoGMPereira/autocut/server/internal/api/handlers"
	"github.com/joaoGMPereira/autocut/server/internal/database"

	_ "modernc.org/sqlite"
)

// openTestDB opens an in-memory SQLite database and applies the migrations.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Apply the minimal schema needed for channels
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS channels (
			id                    INTEGER PRIMARY KEY AUTOINCREMENT,
			name                  TEXT    NOT NULL DEFAULT '',
			channel_id            TEXT    NOT NULL DEFAULT '',
			channel_title         TEXT    NOT NULL DEFAULT '',
			avatar_url            TEXT    NOT NULL DEFAULT '',
			access_token          TEXT    NOT NULL DEFAULT '',
			refresh_token         TEXT    NOT NULL DEFAULT '',
			expires_at            INTEGER NOT NULL DEFAULT 0,
			oauth_client_secret_id INTEGER,
			is_favorite           INTEGER NOT NULL DEFAULT 0,
			created_at            INTEGER NOT NULL DEFAULT 0,
			updated_at            INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE IF NOT EXISTS app_settings (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			key        TEXT    NOT NULL UNIQUE,
			value      TEXT    NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL DEFAULT 0
		);
	`)
	if err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	return db
}

// setupChannelHandler builds a ChannelHandler backed by an in-memory DB.
func setupChannelHandler(t *testing.T) (*handlers.ChannelHandler, *sql.DB) {
	t.Helper()
	db := openTestDB(t)
	return handlers.NewChannelHandler(db), db
}

func TestGetChannelsEmpty(t *testing.T) {
	h, _ := setupChannelHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	rec := httptest.NewRecorder()

	h.GetChannels(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result []database.Channel
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d channels", len(result))
	}
}

func TestCreateChannel(t *testing.T) {
	h, _ := setupChannelHandler(t)

	body := `{"name":"TestChannel","youtube_channel_id":"UCtest123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.PostChannel(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var ch database.Channel
	if err := json.Unmarshal(rec.Body.Bytes(), &ch); err != nil {
		t.Fatalf("decode created channel: %v", err)
	}
	if ch.Name != "TestChannel" {
		t.Errorf("expected name %q, got %q", "TestChannel", ch.Name)
	}

	// Now verify it appears in the list
	req2 := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	rec2 := httptest.NewRecorder()
	h.GetChannels(rec2, req2)

	var list []database.Channel
	if err := json.Unmarshal(rec2.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(list))
	}
	if list[0].Name != "TestChannel" {
		t.Errorf("expected channel name %q, got %q", "TestChannel", list[0].Name)
	}
}

func TestDeleteChannel(t *testing.T) {
	h, _ := setupChannelHandler(t)

	// Create a channel first
	body := `{"name":"ToDelete"}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/channels", bytes.NewBufferString(body))
	postReq.Header.Set("Content-Type", "application/json")
	postRec := httptest.NewRecorder()
	h.PostChannel(postRec, postReq)

	if postRec.Code != http.StatusCreated {
		t.Fatalf("setup: create channel failed: %s", postRec.Body.String())
	}

	var created database.Channel
	if err := json.Unmarshal(postRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}

	// Delete it
	delReq := httptest.NewRequest(http.MethodDelete, "/api/channels/"+itoa64(created.ID), nil)
	delReq.SetPathValue("id", itoa64(created.ID))
	delRec := httptest.NewRecorder()
	h.DeleteChannel(delRec, delReq)

	if delRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", delRec.Code, delRec.Body.String())
	}

	// Verify list is now empty
	listReq := httptest.NewRequest(http.MethodGet, "/api/channels", nil)
	listRec := httptest.NewRecorder()
	h.GetChannels(listRec, listReq)

	var list []database.Channel
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list after delete, got %d", len(list))
	}
}

// itoa64 converts int64 to string (avoids strconv import in test file).
func itoa64(n int64) string {
	return string([]byte(itoa64Str(n)))
}

func itoa64Str(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
