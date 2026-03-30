package uploader

import (
	"testing"
	"time"
)

// mockQuotaRepo is an in-memory QuotaUsageRepository for testing.
type mockQuotaRepo struct {
	recorded []recordedCall
	usage    map[string]int // key: channelID+date string
}

type recordedCall struct {
	channelID string
	operation string
	cost      int
	date      time.Time
}

func (m *mockQuotaRepo) RecordUsage(channelID, operation string, cost int, date time.Time) error {
	m.recorded = append(m.recorded, recordedCall{channelID, operation, cost, date})
	key := channelID + date.Format("2006-01-02")
	m.usage[key] += cost
	return nil
}

func (m *mockQuotaRepo) UsageByDate(channelID string, date time.Time) (int, error) {
	key := channelID + date.Format("2006-01-02")
	return m.usage[key], nil
}

func newMockRepo() *mockQuotaRepo {
	return &mockQuotaRepo{usage: make(map[string]int)}
}

// TestTrack verifies that Track delegates to RecordUsage with the correct arguments.
func TestTrack(t *testing.T) {
	repo := newMockRepo()
	tracker := NewQuotaTracker(repo)

	if err := tracker.Track("ch-001", "upload", QuotaCostUpload); err != nil {
		t.Fatalf("Track: %v", err)
	}

	if len(repo.recorded) != 1 {
		t.Fatalf("expected 1 recorded call, got %d", len(repo.recorded))
	}

	call := repo.recorded[0]
	if call.channelID != "ch-001" {
		t.Errorf("channelID: got %q, want %q", call.channelID, "ch-001")
	}
	if call.operation != "upload" {
		t.Errorf("operation: got %q, want %q", call.operation, "upload")
	}
	if call.cost != QuotaCostUpload {
		t.Errorf("cost: got %d, want %d", call.cost, QuotaCostUpload)
	}
}

// TestRemaining verifies that Remaining returns 10 000 - usageByDate.
// With 3 000 units used, 7 000 should remain.
func TestRemaining(t *testing.T) {
	repo := newMockRepo()
	// Pre-seed 3 000 units consumed today for "ch-002".
	today := time.Now().Format("2006-01-02")
	repo.usage["ch-002"+today] = 3000

	tracker := NewQuotaTracker(repo)
	remaining, err := tracker.Remaining("ch-002")
	if err != nil {
		t.Fatalf("Remaining: %v", err)
	}

	const want = 7000
	if remaining != want {
		t.Errorf("Remaining: got %d, want %d", remaining, want)
	}
}
