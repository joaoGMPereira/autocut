package downloader

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// thumbnailIndex is the on-disk cache manifest: url → local filename.
type thumbnailIndex map[string]string

// ThumbnailCache caches downloaded thumbnails both in memory and on disk.
// Kotlin ref: ThumbnailDownloadDelegate (session cache + disk check).
type ThumbnailCache struct {
	cacheDir string
	mu       sync.RWMutex
	index    thumbnailIndex // url → absolute local path
	log      *slog.Logger
}

// indexPath returns the absolute path to cacheDir/index.json.
func (c *ThumbnailCache) indexPath() string {
	return filepath.Join(c.cacheDir, "index.json")
}

// NewThumbnailCache creates a ThumbnailCache backed by cacheDir.
// Loads an existing index.json if present.
func NewThumbnailCache(cacheDir string) *ThumbnailCache {
	tc := &ThumbnailCache{
		cacheDir: cacheDir,
		index:    make(thumbnailIndex),
		log:      slog.With("component", "downloader", "op", "thumbnail-cache"),
	}
	tc.loadIndex()
	return tc
}

// Get returns the local file path for url if present in the cache.
func (c *ThumbnailCache) Get(url string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	path, ok := c.index[url]
	return path, ok
}

// Put stores a url → localPath mapping in memory and persists to index.json.
func (c *ThumbnailCache) Put(url, localPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.index[url] = localPath
	c.saveIndex()
}

// Download returns the local path for url, downloading it first when not cached.
// The file is saved as cacheDir/<sha256(url)>.jpg.
func (c *ThumbnailCache) Download(url, cacheDir string) (string, error) {
	if path, ok := c.Get(url); ok {
		// Verify file still exists on disk.
		if _, err := os.Stat(path); err == nil {
			c.log.Debug("thumbnail cache hit", "url", url, "path", path)
			return path, nil
		}
		// Stale entry — remove and re-download.
		c.mu.Lock()
		delete(c.index, url)
		c.saveIndex()
		c.mu.Unlock()
	}

	localPath, err := c.downloadToDir(url, cacheDir)
	if err != nil {
		return "", err
	}
	c.Put(url, localPath)
	return localPath, nil
}

// downloadToDir fetches url via HTTP and saves it to dir/<sha256(url)>.jpg.
func (c *ThumbnailCache) downloadToDir(url, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	hash := sha256.Sum256([]byte(url))
	filename := fmt.Sprintf("%x.jpg", hash)
	destPath := filepath.Join(dir, filename)

	c.log.Info("downloading thumbnail", "url", url, "dest", destPath)

	resp, err := http.Get(url) //nolint:gosec // URL comes from yt-dlp metadata
	if err != nil {
		return "", fmt.Errorf("http get thumbnail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("thumbnail http %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create thumbnail file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("write thumbnail: %w", err)
	}

	c.log.Info("thumbnail saved", "path", destPath)
	return destPath, nil
}

// loadIndex reads cacheDir/index.json into memory (best-effort, ignores missing file).
func (c *ThumbnailCache) loadIndex() {
	data, err := os.ReadFile(c.indexPath())
	if err != nil {
		return // file does not exist yet — that is fine
	}
	if err := json.Unmarshal(data, &c.index); err != nil {
		c.log.Warn("corrupt thumbnail index, starting fresh", "err", err)
		c.index = make(thumbnailIndex)
	}
}

// saveIndex persists the current index to cacheDir/index.json.
// Must be called with c.mu write-locked.
func (c *ThumbnailCache) saveIndex() {
	if err := os.MkdirAll(c.cacheDir, 0o755); err != nil {
		c.log.Error("cannot create cache dir", "err", err)
		return
	}
	data, err := json.MarshalIndent(c.index, "", "  ")
	if err != nil {
		c.log.Error("marshal thumbnail index", "err", err)
		return
	}
	if err := os.WriteFile(c.indexPath(), data, 0o644); err != nil {
		c.log.Error("write thumbnail index", "err", err)
	}
}
