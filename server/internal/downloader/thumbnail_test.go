package downloader

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestThumbnailCache_GetMiss(t *testing.T) {
	cache := NewThumbnailCache(t.TempDir())
	_, ok := cache.Get("https://example.com/thumb.jpg")
	if ok {
		t.Fatal("expected cache miss, got hit")
	}
}

func TestThumbnailCache_PutThenGet(t *testing.T) {
	dir := t.TempDir()
	cache := NewThumbnailCache(dir)

	const url = "https://example.com/thumb.jpg"
	const localPath = "/tmp/thumb.jpg"

	cache.Put(url, localPath)

	got, ok := cache.Get(url)
	if !ok {
		t.Fatal("expected cache hit after Put")
	}
	if got != localPath {
		t.Errorf("path mismatch: want %q, got %q", localPath, got)
	}
}

func TestThumbnailCache_PutPersistsToIndexJSON(t *testing.T) {
	dir := t.TempDir()
	cache := NewThumbnailCache(dir)

	const url = "https://example.com/persist.jpg"
	cache.Put(url, "/some/local/path.jpg")

	// Load a fresh cache from the same directory — it must see the entry.
	cache2 := NewThumbnailCache(dir)
	got, ok := cache2.Get(url)
	if !ok {
		t.Fatal("expected entry to survive index reload")
	}
	if got != "/some/local/path.jpg" {
		t.Errorf("persisted path mismatch: got %q", got)
	}
}

func TestThumbnailCache_Download_DownloadsAndCaches(t *testing.T) {
	// Serve fake JPEG bytes.
	fakeImage := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10} // minimal JPEG header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeImage)
	}))
	defer srv.Close()

	dir := t.TempDir()
	cache := NewThumbnailCache(dir)

	path, err := cache.Download(srv.URL+"/thumb.jpg", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File must exist on disk.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("downloaded file not readable: %v", err)
	}
	if string(data) != string(fakeImage) {
		t.Errorf("file content mismatch")
	}

	// Must be cached now — second call should return same path without HTTP.
	path2, err := cache.Download(srv.URL+"/thumb.jpg", dir)
	if err != nil {
		t.Fatalf("second download error: %v", err)
	}
	if path2 != path {
		t.Errorf("cached path mismatch: want %q, got %q", path, path2)
	}
}

func TestThumbnailCache_Download_StaleEntryRedownloads(t *testing.T) {
	fakeImage := []byte("fake-image-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeImage)
	}))
	defer srv.Close()

	dir := t.TempDir()
	cache := NewThumbnailCache(dir)

	// Insert a stale entry pointing to a non-existent file.
	cache.Put(srv.URL+"/stale.jpg", filepath.Join(dir, "ghost.jpg"))

	path, err := cache.Download(srv.URL+"/stale.jpg", dir)
	if err != nil {
		t.Fatalf("unexpected error on stale re-download: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("re-downloaded file should exist: %v", err)
	}
}
