package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ClaudeCLI wraps the `claude` CLI binary for AI text generation.
type ClaudeCLI struct {
	binaryPath string
}

// NewClaudeCLI discovers the claude CLI binary on the system.
// Returns an error if not found.
func NewClaudeCLI() (*ClaudeCLI, error) {
	// Attempt 1: standard PATH lookup
	if p, err := exec.LookPath("claude"); err == nil {
		return &ClaudeCLI{binaryPath: p}, nil
	}

	// Attempt 2: shell login discovery — for GUI apps (Electron) on macOS/Linux
	// that inherit a minimal PATH from launchd/systemd.
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/zsh"
		}
		cmd := exec.Command(shell, "-lc", "which claude 2>/dev/null || command -v claude 2>/dev/null")
		cmd.Env = append(os.Environ(), "TERM=dumb")
		if out, err := cmd.Output(); err == nil {
			if p := strings.TrimSpace(string(out)); p != "" {
				slog.Info("claude CLI found via shell discovery", "path", p)
				return &ClaudeCLI{binaryPath: p}, nil
			}
		}
	}

	return nil, fmt.Errorf("claude CLI not found: install with 'npm install -g @anthropic-ai/claude-code'")
}

// Generate runs the claude CLI and returns the final result text.
func (c *ClaudeCLI) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	return c.GenerateStream(ctx, req, nil)
}

// GenerateStream runs the claude CLI, calling onChunk with each text delta for progress.
// Returns the final accumulated result text.
func (c *ClaudeCLI) GenerateStream(ctx context.Context, req GenerateRequest, onChunk func(delta string)) (string, error) {
	model := req.Model
	if model == "" {
		model = "haiku"
	}

	args := []string{
		"-p", req.Prompt,
		"--output-format", "stream-json",
		"--model", model,
	}
	if req.SystemPrompt != "" {
		args = append(args, "--system-prompt", req.SystemPrompt)
	}

	cmd := exec.CommandContext(ctx, c.binaryPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("claude: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("claude: start: %w", err)
	}

	var result strings.Builder

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			slog.Debug("claude: skipping non-JSON line", "line", line[:min(len(line), 100)])
			continue
		}

		typ, _ := raw["type"].(string)
		switch typ {
		case "assistant":
			// Extract text from message.content[] blocks
			if msg, ok := raw["message"].(map[string]any); ok {
				if content, ok := msg["content"].([]any); ok {
					for _, item := range content {
						block, ok := item.(map[string]any)
						if !ok {
							continue
						}
						if blockType, _ := block["type"].(string); blockType == "text" {
							if text, _ := block["text"].(string); text != "" {
								result.WriteString(text)
								if onChunk != nil {
									onChunk(text)
								}
							}
						}
					}
				}
			}

		case "result":
			// The result field contains the final complete text
			if resultText, ok := raw["result"].(string); ok && resultText != "" {
				// Result replaces accumulated text (it's the complete response)
				result.Reset()
				result.WriteString(resultText)
			}
		}
		// Skip "system", "ping", etc.
	}

	if err := scanner.Err(); err != nil {
		slog.Warn("claude: scanner error", "err", err)
	}

	if err := cmd.Wait(); err != nil {
		return result.String(), fmt.Errorf("claude: process exited with error: %w", err)
	}

	return result.String(), nil
}
