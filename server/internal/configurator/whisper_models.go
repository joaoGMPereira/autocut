package configurator

import (
	"context"
	"fmt"
	"os"
)

// whisperModel holds metadata for a known Whisper model variant.
type whisperModel struct {
	name     string // e.g. "tiny"
	filename string // e.g. "ggml-tiny.bin"
	sizeMB   int
}

var knownWhisperModels = []whisperModel{
	{name: "tiny", filename: "ggml-tiny.bin", sizeMB: 75},
	{name: "base", filename: "ggml-base.bin", sizeMB: 148},
	{name: "small", filename: "ggml-small.bin", sizeMB: 488},
	{name: "medium", filename: "ggml-medium.bin", sizeMB: 1500},
	{name: "large", filename: "ggml-large.bin", sizeMB: 3100},
}

const whisperBaseURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/"

// WhisperModelStatus returns download status for all known Whisper models.
func WhisperModelStatus(dir *AutoCutDir) []WhisperModelInfo {
	result := make([]WhisperModelInfo, len(knownWhisperModels))
	for i, m := range knownWhisperModels {
		path := dir.WhisperModelPath(m.filename)
		_, err := os.Stat(path)
		downloaded := err == nil
		info := WhisperModelInfo{
			Name:       m.name,
			Downloaded: downloaded,
			SizeMB:     m.sizeMB,
		}
		if downloaded {
			info.Path = path
		}
		result[i] = info
	}
	return result
}

// DownloadWhisperModel downloads the named Whisper model into dir.WhisperDir.
// Logs progress to logCh.
func DownloadWhisperModel(ctx context.Context, name string, dir *AutoCutDir, logCh chan<- string) error {
	var model *whisperModel
	for i := range knownWhisperModels {
		if knownWhisperModels[i].name == name {
			model = &knownWhisperModels[i]
			break
		}
	}
	if model == nil {
		return fmt.Errorf("unknown whisper model: %s", name)
	}

	if err := os.MkdirAll(dir.WhisperDir, 0o755); err != nil {
		return fmt.Errorf("mkdir whisper dir: %w", err)
	}

	dest := dir.WhisperModelPath(model.filename)
	url := whisperBaseURL + model.filename
	logCh <- fmt.Sprintf("Downloading Whisper model %q (%d MB) from %s ...", name, model.sizeMB, url)

	if err := downloadFile(ctx, url, dest, logCh); err != nil {
		return fmt.Errorf("download whisper model %s: %w", name, err)
	}

	logCh <- fmt.Sprintf("Whisper model %q downloaded to %s", name, dest)
	return nil
}
