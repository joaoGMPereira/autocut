// server/internal/logsink/logsink_test.go
package logsink

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func newTestSink() *LogSink {
	return New(io.Discard)
}

func makeRecord(msg string, level slog.Level) slog.Record {
	return slog.NewRecord(time.Now(), level, msg, 0)
}

func TestHandle_BuffersEntry(t *testing.T) {
	s := newTestSink()
	r := makeRecord("hello", slog.LevelInfo)
	_ = s.Handle(context.Background(), r)

	hist := s.History()
	if len(hist) != 1 {
		t.Fatalf("want 1 entry, got %d", len(hist))
	}
	if hist[0].Msg != "hello" {
		t.Errorf("want msg=hello, got %q", hist[0].Msg)
	}
	if hist[0].Level != "info" {
		t.Errorf("want level=info, got %q", hist[0].Level)
	}
	if hist[0].Source != "go" {
		t.Errorf("want source=go, got %q", hist[0].Source)
	}
}

func TestHandle_BroadcastsToListener(t *testing.T) {
	s := newTestSink()
	ch, cancel := s.Register()
	defer cancel()

	r := makeRecord("broadcast", slog.LevelWarn)
	_ = s.Handle(context.Background(), r)

	select {
	case entry := <-ch:
		if entry.Msg != "broadcast" {
			t.Errorf("want msg=broadcast, got %q", entry.Msg)
		}
		if entry.Level != "warn" {
			t.Errorf("want level=warn, got %q", entry.Level)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestHandle_RingBuffer_Evicts(t *testing.T) {
	s := newTestSink()
	for i := 0; i < bufCap+10; i++ {
		r := makeRecord("msg", slog.LevelInfo)
		_ = s.Handle(context.Background(), r)
	}
	hist := s.History()
	if len(hist) != bufCap {
		t.Fatalf("want buf len=%d, got %d", bufCap, len(hist))
	}
}

func TestRegister_ReplaysBuf(t *testing.T) {
	s := newTestSink()
	for _, msg := range []string{"a", "b", "c"} {
		r := makeRecord(msg, slog.LevelInfo)
		_ = s.Handle(context.Background(), r)
	}

	ch, cancel := s.Register()
	defer cancel()

	var got []string
	for i := 0; i < 3; i++ {
		select {
		case e := <-ch:
			got = append(got, e.Msg)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("timeout at entry %d", i)
		}
	}
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("unexpected replay: %v", got)
	}
}

func TestHistory_ReturnsCopy(t *testing.T) {
	s := newTestSink()
	r := makeRecord("x", slog.LevelInfo)
	_ = s.Handle(context.Background(), r)

	h1 := s.History()
	h1[0].Msg = "mutated"
	h2 := s.History()
	if h2[0].Msg == "mutated" {
		t.Error("History should return a copy, not a reference")
	}
}
