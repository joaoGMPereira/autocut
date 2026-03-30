package transcript

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const hashReadLimit = 1 << 20 // 1 MB

// TranscriptCache stores and retrieves Transcript values as JSON files.
// The cache key is a SHA-256 hash of the first 1 MB of the source file plus
// its total size, so re-transcribing an unchanged file is cheap.
//
// Kotlin ref: TranscriptCacheDelegate (uses videoId as key; Go uses file hash)
type TranscriptCache struct {
	cacheDir string
	log      *slog.Logger
}

// NewCache creates a TranscriptCache that writes files to cacheDir.
// The directory is created if it does not exist.
func NewCache(cacheDir string) *TranscriptCache {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		slog.Warn("transcript cache: could not create cache dir",
			"component", "transcript", "dir", cacheDir, "err", err)
	}
	return &TranscriptCache{
		cacheDir: cacheDir,
		log:      slog.With("component", "transcript"),
	}
}

// Hash returns a deterministic hex string for the file at filePath.
// It reads at most 1 MB from the file and combines that with the file's total
// size, so renames and appends are treated as distinct inputs.
func (c *TranscriptCache) Hash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("transcript cache hash: open %q: %w", filePath, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("transcript cache hash: stat %q: %w", filePath, err)
	}

	h := sha256.New()
	if _, err := io.CopyN(h, f, hashReadLimit); err != nil && err != io.EOF {
		return "", fmt.Errorf("transcript cache hash: read %q: %w", filePath, err)
	}

	// Mix in total file size to distinguish truncated vs. full files.
	_, _ = fmt.Fprintf(h, ":%d", stat.Size())

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Get returns the cached Transcript for videoHash, or (nil, false) on miss.
func (c *TranscriptCache) Get(videoHash string) (*Transcript, bool) {
	path := c.cachePath(videoHash)
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			c.log.Warn("transcript cache get: unexpected read error",
				"op", "transcript", "hash", videoHash, "err", err)
		}
		return nil, false
	}

	var t Transcript
	if err := json.Unmarshal(data, &t); err != nil {
		c.log.Warn("transcript cache get: corrupt entry, ignoring",
			"op", "transcript", "hash", videoHash, "err", err)
		return nil, false
	}

	c.log.Info("transcript cache hit", "op", "transcript", "hash", videoHash)
	return &t, true
}

// Put serialises t as JSON and writes it to cacheDir/{videoHash}.json.
func (c *TranscriptCache) Put(videoHash string, t *Transcript) error {
	if err := os.MkdirAll(c.cacheDir, 0o755); err != nil {
		return fmt.Errorf("transcript cache put: mkdir: %w", err)
	}

	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("transcript cache put: marshal: %w", err)
	}

	path := c.cachePath(videoHash)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("transcript cache put: write %q: %w", path, err)
	}

	c.log.Info("transcript cache stored", "op", "transcript", "hash", videoHash, "path", path)
	return nil
}

func (c *TranscriptCache) cachePath(hash string) string {
	return filepath.Join(c.cacheDir, hash+".json")
}
