// server/internal/logsink/logsink.go
package logsink

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

const bufCap = 500

// LogEntry is the JSON shape sent to the frontend.
type LogEntry struct {
	ID     string                 `json:"id"`
	Ts     string                 `json:"ts"`
	Level  string                 `json:"level"`
	Source string                 `json:"source"`
	Msg    string                 `json:"msg"`
	Attrs  map[string]interface{} `json:"attrs,omitempty"`
}

// sharedState is held by pointer so WithAttrs/WithGroup clones share it.
type sharedState struct {
	mu      sync.Mutex
	buf     []LogEntry
	clients []chan LogEntry
	seq     uint64
}

// LogSink is a slog.Handler that tees to an inner handler, buffers entries,
// and broadcasts to registered SSE listeners.
type LogSink struct {
	inner  slog.Handler
	shared *sharedState
}

// New creates a LogSink that writes text logs to w (stderr in production).
func New(w io.Writer) *LogSink {
	return &LogSink{
		inner: slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}),
		shared: &sharedState{
			buf: make([]LogEntry, 0, bufCap),
		},
	}
}

func (s *LogSink) Enabled(ctx context.Context, level slog.Level) bool {
	return s.inner.Enabled(ctx, level)
}

func (s *LogSink) Handle(ctx context.Context, r slog.Record) error {
	// Always write to stderr via inner handler; ignore its error.
	_ = s.inner.Handle(ctx, r)

	attrs := make(map[string]interface{})
	r.Attrs(func(a slog.Attr) bool {
		v := a.Value.Resolve().Any()
		if err, ok := v.(error); ok {
			v = err.Error()
		}
		attrs[a.Key] = v
		return true
	})
	if len(attrs) == 0 {
		attrs = nil
	}

	s.shared.mu.Lock()
	s.shared.seq++
	entry := LogEntry{
		ID:     fmt.Sprintf("%d", s.shared.seq),
		Ts:     r.Time.UTC().Format(time.RFC3339Nano),
		Level:  levelName(r.Level),
		Source: "go",
		Msg:    r.Message,
		Attrs:  attrs,
	}
	if len(s.shared.buf) >= bufCap {
		s.shared.buf = append(s.shared.buf[1:], entry)
	} else {
		s.shared.buf = append(s.shared.buf, entry)
	}
	clients := make([]chan LogEntry, len(s.shared.clients))
	copy(clients, s.shared.clients)
	s.shared.mu.Unlock()

	for _, ch := range clients {
		select {
		case ch <- entry:
		default:
		}
	}
	return nil
}

func (s *LogSink) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogSink{inner: s.inner.WithAttrs(attrs), shared: s.shared}
}

func (s *LogSink) WithGroup(name string) slog.Handler {
	return &LogSink{inner: s.inner.WithGroup(name), shared: s.shared}
}

// Register adds a listener. Returns a receive channel pre-loaded with the
// replay buffer and a cancel func that must be called on disconnect.
func (s *LogSink) Register() (<-chan LogEntry, func()) {
	ch := make(chan LogEntry, 128)

	s.shared.mu.Lock()
	for _, e := range s.shared.buf {
		select {
		case ch <- e:
		default:
		}
	}
	s.shared.clients = append(s.shared.clients, ch)
	s.shared.mu.Unlock()

	cancel := func() {
		s.shared.mu.Lock()
		defer s.shared.mu.Unlock()
		for i, c := range s.shared.clients {
			if c == ch {
				s.shared.clients = append(s.shared.clients[:i], s.shared.clients[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, cancel
}

// History returns a snapshot of the ring buffer.
func (s *LogSink) History() []LogEntry {
	s.shared.mu.Lock()
	defer s.shared.mu.Unlock()
	out := make([]LogEntry, len(s.shared.buf))
	copy(out, s.shared.buf)
	return out
}

func levelName(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "error"
	case l >= slog.LevelWarn:
		return "warn"
	case l >= slog.LevelInfo:
		return "info"
	default:
		return "debug"
	}
}
