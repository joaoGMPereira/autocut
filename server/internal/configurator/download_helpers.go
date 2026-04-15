package configurator

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// resolveGitHubLatestTag returns the tag_name of the latest release.
func resolveGitHubLatestTag(ctx context.Context, apiURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode GitHub API response: %w", err)
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

// resolveGitHubAssetURL calls the GitHub releases API endpoint and returns the
// browser_download_url for the first asset whose name contains assetPattern.
func resolveGitHubAssetURL(ctx context.Context, apiURL, assetPattern string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode GitHub API response: %w", err)
	}

	for _, a := range release.Assets {
		if strings.Contains(a.Name, assetPattern) {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no release asset matching %q found", assetPattern)
}

// unzipSingleBinary extracts the binary named binaryName from zipPath to dest.
// If no entry matches binaryName exactly, the first executable-looking entry is used.
func unzipSingleBinary(zipPath, binaryName, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name != binaryName {
			continue
		}
		return extractZipEntry(f, dest)
	}

	// Fallback: first non-directory entry
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			return extractZipEntry(f, dest)
		}
	}
	return fmt.Errorf("no usable entry found in %s", zipPath)
}

func extractZipEntry(f *zip.File, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil { //nolint:gosec
		return err
	}
	return nil
}
