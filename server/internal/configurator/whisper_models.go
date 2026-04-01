package configurator

import (
	"context"
	"fmt"
	"os"
)

// WhisperModelInfo describes a known whisper.cpp model.
type WhisperModelInfo struct {
	Name       string `json:"name"`
	SizeMB     int    `json:"size_mb"`
	Downloaded bool   `json:"downloaded"`
	Path       string `json:"path"`
}

// knownWhisperModels lists the 5 standard whisper.cpp GGML models.
var knownWhisperModels = []struct {
	Name   string
	SizeMB int
}{
	{"tiny", 75}, {"base", 142}, {"small", 466}, {"medium", 1500}, {"large", 2900},
}

// WhisperModelStatus returns download status for all known models.
func (c *Configurator) WhisperModelStatus() []WhisperModelInfo {
	models := make([]WhisperModelInfo, len(knownWhisperModels))
	for i, m := range knownWhisperModels {
		filename := fmt.Sprintf("ggml-%s.bin", m.Name)
		p := c.dir.WhisperModelPath(filename)
		_, err := os.Stat(p)
		downloaded := err == nil
		path := ""
		if downloaded {
			path = p
		}
		models[i] = WhisperModelInfo{
			Name:       m.Name,
			SizeMB:     m.SizeMB,
			Downloaded: downloaded,
			Path:       path,
		}
	}
	return models
}

// DownloadWhisperModel downloads the named model to WhisperDir, streaming progress to logCh.
func (c *Configurator) DownloadWhisperModel(ctx context.Context, model string, logCh chan<- string) error {
	valid := false
	for _, m := range knownWhisperModels {
		if m.Name == model {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unknown whisper model: %s", model)
	}

	filename := fmt.Sprintf("ggml-%s.bin", model)
	url := fmt.Sprintf("https://huggingface.co/ggerganov/whisper.cpp/resolve/main/%s", filename)
	dest := c.dir.WhisperModelPath(filename)

	return downloadFile(ctx, url, dest, logCh)
}
