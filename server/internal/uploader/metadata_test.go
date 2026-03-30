package uploader

import (
	"context"
	"testing"
	"time"
)

// mockYouTubeUploader is a test double that records calls to SetMetadata and PinComment.
type mockYouTubeUploader struct {
	setMetadataCalls []VideoMetadata
	pinCommentCalls  []pinCommentCall
}

type pinCommentCall struct {
	videoID string
	text    string
}

func (m *mockYouTubeUploader) Upload(_ context.Context, _ UploadRequest) (<-chan UploadProgress, error) {
	ch := make(chan UploadProgress, 1)
	ch <- UploadProgress{Done: true}
	close(ch)
	return ch, nil
}

func (m *mockYouTubeUploader) UploadThumbnail(_, _ string) error { return nil }

func (m *mockYouTubeUploader) AddToPlaylist(_, _ string) error { return nil }

func (m *mockYouTubeUploader) Schedule(_ string, _ time.Time) error { return nil }

func (m *mockYouTubeUploader) SetMetadata(_ string, meta VideoMetadata) error {
	m.setMetadataCalls = append(m.setMetadataCalls, meta)
	return nil
}

func (m *mockYouTubeUploader) GetComments(_ string) ([]Comment, error) { return nil, nil }

func (m *mockYouTubeUploader) PinComment(videoID, text string) error {
	m.pinCommentCalls = append(m.pinCommentCalls, pinCommentCall{videoID, text})
	return nil
}

// TestBulkUpdate verifies that BulkUpdate calls SetMetadata once per video.
func TestBulkUpdate(t *testing.T) {
	mock := &mockYouTubeUploader{}
	mgr := NewMetadataManager(mock)

	videos := []VideoMetadata{
		{VideoID: "v1", Title: "Title 1"},
		{VideoID: "v2", Title: "Title 2"},
		{VideoID: "v3", Title: "Title 3"},
	}

	if err := mgr.BulkUpdate(videos); err != nil {
		t.Fatalf("BulkUpdate: %v", err)
	}

	if len(mock.setMetadataCalls) != 3 {
		t.Fatalf("SetMetadata called %d times, want 3", len(mock.setMetadataCalls))
	}
	for i, v := range videos {
		if mock.setMetadataCalls[i].VideoID != v.VideoID {
			t.Errorf("call[%d] VideoID: got %q, want %q", i, mock.setMetadataCalls[i].VideoID, v.VideoID)
		}
	}
}

// TestSyncPinnedComment verifies template variable substitution and that
// PinComment is called with the resolved text.
func TestSyncPinnedComment(t *testing.T) {
	mock := &mockYouTubeUploader{}
	mgr := NewMetadataManager(mock)

	const (
		videoID  = "vid-abc"
		template = "Olá {name}!"
		wantText = "Olá mundo!"
	)

	if err := mgr.SyncPinnedComment(videoID, template, map[string]string{"name": "mundo"}); err != nil {
		t.Fatalf("SyncPinnedComment: %v", err)
	}

	if len(mock.pinCommentCalls) != 1 {
		t.Fatalf("PinComment called %d times, want 1", len(mock.pinCommentCalls))
	}

	call := mock.pinCommentCalls[0]
	if call.videoID != videoID {
		t.Errorf("videoID: got %q, want %q", call.videoID, videoID)
	}
	if call.text != wantText {
		t.Errorf("text: got %q, want %q", call.text, wantText)
	}
}
