package configurator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Configurator is the facade over all ToolValidator instances.
type Configurator struct {
	dir        *AutoCutDir
	validators []ToolValidator
}

// New creates a Configurator with the default validators in canonical order:
// YtDlp → TwitchCLI → FFmpeg → Whisper → Ollama → ImageMagick → Brew.
func New(dir *AutoCutDir) *Configurator {
	return &Configurator{
		dir: dir,
		validators: []ToolValidator{
			NewYtDlpValidator(dir),
			NewTwitchCLIValidator(dir),
			NewFFmpegValidator(dir),
			NewWhisperValidator(dir),
			NewOllamaValidator(dir),
			NewImageMagickValidator(dir),
			NewBrewValidator(),
		},
	}
}

// newWithValidators creates a Configurator with a custom validator list.
// Used in tests to inject mocks.
func newWithValidators(dir *AutoCutDir, validators []ToolValidator) *Configurator {
	return &Configurator{dir: dir, validators: validators}
}

// Status returns the ToolStatus of every registered validator.
func (c *Configurator) Status() []ToolStatus {
	out := make([]ToolStatus, len(c.validators))
	for i, v := range c.validators {
		out[i] = v.Status()
	}
	return out
}

// Required returns the status of every required tool.
func (c *Configurator) Required() []ToolStatus {
	var out []ToolStatus
	for _, v := range c.validators {
		if s := v.Status(); s.Required {
			out = append(out, s)
		}
	}
	return out
}

// Missing returns the status of every tool that is not installed.
func (c *Configurator) Missing() []ToolStatus {
	var out []ToolStatus
	for _, v := range c.validators {
		if s := v.Status(); !s.Installed {
			out = append(out, s)
		}
	}
	return out
}

// Get returns the validator with the given name, or (nil, false) if not found.
func (c *Configurator) Get(name string) (ToolValidator, bool) {
	for _, v := range c.validators {
		if v.Name() == name {
			return v, true
		}
	}
	return nil, false
}

// Install delegates to the named validator's Install method.
func (c *Configurator) Install(ctx context.Context, name string, logCh chan<- string) error {
	v, ok := c.Get(name)
	if !ok {
		return &ErrToolNotFound{Name: name}
	}
	return v.Install(ctx, logCh)
}

// ResolvedPaths returns a map of tool name → resolved path for every validator.
// Tools that are not installed map to an empty string.
func (c *Configurator) ResolvedPaths() map[string]string {
	out := make(map[string]string, len(c.validators))
	for _, v := range c.validators {
		out[v.Name()] = v.ResolvedPath()
	}
	return out
}

// AllInstalled returns true when every required tool is installed.
func (c *Configurator) AllInstalled() bool {
	for _, v := range c.validators {
		if s := v.Status(); s.Required && !s.Installed {
			return false
		}
	}
	return true
}

// Dir returns the AutoCutDir used by this Configurator.
func (c *Configurator) Dir() *AutoCutDir {
	return c.dir
}

// UpdateInfo holds the version comparison result for a tool.
type UpdateInfo struct {
	Name            string `json:"name"`
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
}

// githubRepos maps tool names to their GitHub owner/repo for update checks.
var githubRepos = map[string]string{
	"yt-dlp":              "yt-dlp/yt-dlp",
	"TwitchDownloaderCLI": "lay295/TwitchDownloader",
}

// CheckUpdate returns version information for the named tool.
// For tools with a known GitHub repo, it also fetches the latest release tag.
func (c *Configurator) CheckUpdate(ctx context.Context, name string) (UpdateInfo, error) {
	v, ok := c.Get(name)
	if !ok {
		return UpdateInfo{}, &ErrToolNotFound{Name: name}
	}

	info := UpdateInfo{
		Name:           name,
		CurrentVersion: v.Version(),
	}

	repo, hasRepo := githubRepos[name]
	if hasRepo {
		tag, err := fetchLatestGitHubTag(ctx, repo)
		if err == nil {
			info.LatestVersion = tag
		}
	}

	info.UpdateAvailable = info.LatestVersion != "" &&
		info.CurrentVersion != "" &&
		info.CurrentVersion != info.LatestVersion

	return info, nil
}

// fetchLatestGitHubTag fetches the tag_name from a GitHub repo's latest release.
func fetchLatestGitHubTag(ctx context.Context, repo string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github: status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

// ErrToolNotFound is returned when Install is called with an unknown tool name.
type ErrToolNotFound struct {
	Name string
}

func (e *ErrToolNotFound) Error() string {
	return "configurator: tool not found: " + e.Name
}
