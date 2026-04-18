package ai

import (
	"encoding/json"
	"testing"
)

// TestComputeAnalytics_Empty verifies that an empty video list produces zero-valued
// analytics with JSON "[]" placeholders for all slice fields.
func TestComputeAnalytics_Empty(t *testing.T) {
	got := computeAnalytics(42, nil)

	if got.VideoCount != 0 {
		t.Errorf("VideoCount: want 0, got %d", got.VideoCount)
	}
	if got.AvgViews != 0 {
		t.Errorf("AvgViews: want 0, got %d", got.AvgViews)
	}
	for name, field := range map[string]string{
		"TopTitleWords": got.TopTitleWords,
		"TopTags":       got.TopTags,
		"SuccessTitles": got.SuccessTitles,
		"RawTitles":     got.RawTitles,
	} {
		if field != "[]" {
			t.Errorf("%s: want \"[]\", got %q", name, field)
		}
	}
	if got.ChannelID != 42 {
		t.Errorf("ChannelID: want 42, got %d", got.ChannelID)
	}
}

// TestComputeAnalytics_StopwordsFiltered verifies that Portuguese and English stopwords
// are excluded from TopTitleWords and that meaningful words are retained.
func TestComputeAnalytics_StopwordsFiltered(t *testing.T) {
	videos := []ytDlpVideo{
		{Title: "o tutorial de programação", ViewCount: 100},
	}
	got := computeAnalytics(1, videos)

	var words []WordCount
	if err := json.Unmarshal([]byte(got.TopTitleWords), &words); err != nil {
		t.Fatalf("TopTitleWords unmarshal: %v", err)
	}

	wordMap := make(map[string]int, len(words))
	for _, wc := range words {
		wordMap[wc.Word] = wc.Count
	}

	// Meaningful words must be present
	if wordMap["tutorial"] == 0 {
		t.Errorf("expected 'tutorial' in top words, got %v", wordMap)
	}
	if wordMap["programação"] == 0 {
		t.Errorf("expected 'programação' in top words, got %v", wordMap)
	}

	// Stopwords must be absent
	for _, sw := range []string{"o", "de"} {
		if _, found := wordMap[sw]; found {
			t.Errorf("stopword %q should not appear in top words", sw)
		}
	}
}

// TestComputeAnalytics_SuccessTitlesSortedByViews verifies that SuccessTitles[0] is
// the title with the highest view count.
func TestComputeAnalytics_SuccessTitlesSortedByViews(t *testing.T) {
	videos := []ytDlpVideo{
		{Title: "low views",    ViewCount: 100},
		{Title: "medium views", ViewCount: 500},
		{Title: "top video",    ViewCount: 9999},
		{Title: "decent views", ViewCount: 300},
		{Title: "ok views",     ViewCount: 200},
	}
	got := computeAnalytics(1, videos)

	var titles []string
	if err := json.Unmarshal([]byte(got.SuccessTitles), &titles); err != nil {
		t.Fatalf("SuccessTitles unmarshal: %v", err)
	}

	if len(titles) == 0 {
		t.Fatal("SuccessTitles is empty")
	}
	if titles[0] != "top video" {
		t.Errorf("SuccessTitles[0]: want \"top video\", got %q", titles[0])
	}
}

// TestComputeAnalytics_AvgViewsRoundsDown verifies integer division truncation for AvgViews.
func TestComputeAnalytics_AvgViewsRoundsDown(t *testing.T) {
	videos := []ytDlpVideo{
		{Title: "video one",   ViewCount: 100},
		{Title: "video two",   ViewCount: 200},
		{Title: "video three", ViewCount: 150},
	}
	got := computeAnalytics(1, videos)

	// (100 + 200 + 150) / 3 = 450 / 3 = 150
	if got.AvgViews != 150 {
		t.Errorf("AvgViews: want 150, got %d", got.AvgViews)
	}
}

// TestComputeAnalytics_TagsAggregated verifies that tags shared across multiple videos
// are counted correctly.
func TestComputeAnalytics_TagsAggregated(t *testing.T) {
	videos := []ytDlpVideo{
		{Title: "video one",   ViewCount: 100, Tags: []string{"javascript", "web"}},
		{Title: "video two",   ViewCount: 200, Tags: []string{"javascript", "react"}},
		{Title: "video three", ViewCount: 150, Tags: []string{"javascript"}},
	}
	got := computeAnalytics(1, videos)

	var tags []TagCount
	if err := json.Unmarshal([]byte(got.TopTags), &tags); err != nil {
		t.Fatalf("TopTags unmarshal: %v", err)
	}

	tagMap := make(map[string]int, len(tags))
	for _, tc := range tags {
		tagMap[tc.Tag] = tc.Count
	}

	if tagMap["javascript"] != 3 {
		t.Errorf("tag 'javascript': want count 3, got %d (tagMap=%v)", tagMap["javascript"], tagMap)
	}
}
