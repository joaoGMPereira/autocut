package downloader

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"
)

// ThumbnailCache maps remote thumbnail URLs to local file paths.
// The in-memory map is the hot path; index.json is the persistence layer.
type ThumbnailCache struct {
	dir     string
	mu      sync.Mutex
	entries map[string]string
}

// NewThumbnailCache creates a cache rooted at dir.
// If dir/index.json already exists its entries are loaded into memory.
func NewThumbnailCache(dir string) *ThumbnailCache {
	c := &ThumbnailCache{
		dir:     dir,
		entries: make(map[string]string),
	}
	indexPath := filepath.Join(dir, "index.json")
	if data, err := os.ReadFile(indexPath); err == nil {
		_ = json.Unmarshal(data, &c.entries)
	}
	return c
}

// Get looks up a URL in the in-memory map (pre-loaded from index.json).
// Returns the local path and true on a hit; empty string and false on a miss.
func (c *ThumbnailCache) Get(rawURL string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.entries[rawURL]
	return p, ok
}

// Put stores url→localPath in the in-memory map and persists index.json to disk atomically.
func (c *ThumbnailCache) Put(rawURL, localPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[rawURL] = localPath
	data, err := json.Marshal(c.entries)
	if err != nil {
		return
	}
	tmpFile := filepath.Join(c.dir, "index.json.tmp")
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmpFile, filepath.Join(c.dir, "index.json"))
}

// Download returns the local path for rawURL, downloading the image if necessary.
//
// Logic:
//  1. Call Get(rawURL).
//  2. If hit AND the file exists on disk → return cached path (no HTTP).
//  3. If hit BUT file is missing (stale entry) → fall through to download.
//  4. Issue an HTTP GET, write the body to a file in dir, call Put, return path.
func (c *ThumbnailCache) Download(rawURL, dir string) (string, error) {
	if p, ok := c.Get(rawURL); ok {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	resp, err := http.Get(rawURL) //nolint:gosec
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	u, _ := url.Parse(rawURL)
	filename := path.Base(u.Path)
	if filename == "" || filename == "." {
		filename = "thumbnail.jpg"
	}

	destPath := filepath.Join(dir, filename)
	f, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}

	c.Put(rawURL, destPath)
	return destPath, nil
}
