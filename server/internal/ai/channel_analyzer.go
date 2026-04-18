package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/joaoGMPereira/autocut/server/internal/database"
)

// ytDlpVideo holds the fields we parse from each yt-dlp JSON line.
type ytDlpVideo struct {
	Title     string   `json:"title"`
	ViewCount int      `json:"view_count"`
	Tags      []string `json:"tags"`
}

// WordCount is a word/tag frequency pair stored in channel_analytics JSON columns.
type WordCount struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// TagCount is a tag frequency pair.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// ptStopwords is a minimal Portuguese + common English stopword set for title tokenisation.
var ptStopwords = map[string]bool{
	"o": true, "a": true, "e": true, "de": true, "da": true, "do": true,
	"em": true, "um": true, "uma": true, "para": true, "por": true,
	"com": true, "que": true, "se": true, "na": true, "no": true,
	"os": true, "as": true, "é": true, "ao": true, "dos": true,
	"das": true, "nos": true, "nas": true, "the": true, "and": true,
	"of": true, "to": true, "in": true, "is": true, "it": true,
	"this": true, "my": true, "i": true, "you": true,
}

const analyticsTTLSec = 86400 // 24 hours

// ChannelAnalyzer fetches and aggregates a YouTube channel's video history.
type ChannelAnalyzer struct {
	ytDlpPath     string
	analyticsRepo *database.ChannelAnalyticsRepo
	log           *slog.Logger
}

// NewChannelAnalyzer constructs a ChannelAnalyzer.
// ytDlpPath is the path to the yt-dlp binary (e.g. "yt-dlp" or "/usr/local/bin/yt-dlp").
func NewChannelAnalyzer(ytDlpPath string, repo *database.ChannelAnalyticsRepo) *ChannelAnalyzer {
	return &ChannelAnalyzer{
		ytDlpPath:     ytDlpPath,
		analyticsRepo: repo,
		log:           slog.With("component", "channel_analyzer"),
	}
}

// FetchAndCache returns cached analytics for channelID if fresh (TTL 24h).
// On cache miss it runs yt-dlp, computes analytics, upserts, and returns the result.
func (a *ChannelAnalyzer) FetchAndCache(ctx context.Context, channelID int64, channelURL string) (*database.ChannelAnalytics, error) {
	// Cache-first: return immediately if fresh data exists.
	cached, err := a.analyticsRepo.Get(ctx, channelID, analyticsTTLSec)
	if err != nil {
		a.log.Warn("channel_analyzer: cache lookup failed, will re-fetch", "err", err)
	}
	if cached != nil {
		a.log.Debug("channel_analyzer: cache hit", "channel_id", channelID)
		return cached, nil
	}

	videos, err := a.fetchVideos(ctx, channelURL)
	if err != nil {
		return nil, fmt.Errorf("channel_analyzer: fetch %s: %w", channelURL, err)
	}

	analytics := computeAnalytics(channelID, videos)

	if err := a.analyticsRepo.Upsert(ctx, analytics); err != nil {
		return nil, fmt.Errorf("channel_analyzer: upsert: %w", err)
	}
	return analytics, nil
}

// fetchVideos calls yt-dlp and returns parsed video metadata.
func (a *ChannelAnalyzer) fetchVideos(ctx context.Context, channelURL string) ([]ytDlpVideo, error) {
	cmd := exec.CommandContext(ctx,
		a.ytDlpPath,
		"--dump-json",
		"--playlist-end", "50",
		"--skip-download",
		"--no-warnings",
		"--quiet",
		channelURL,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start yt-dlp: %w", err)
	}

	var videos []ytDlpVideo
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 256*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var v ytDlpVideo
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			a.log.Debug("channel_analyzer: skip non-video line", "len", len(line))
			continue
		}
		if v.Title != "" {
			videos = append(videos, v)
		}
	}
	if err := scanner.Err(); err != nil {
		a.log.Warn("channel_analyzer: scanner error", "err", err)
	}
	if err := cmd.Wait(); err != nil {
		return videos, fmt.Errorf("yt-dlp exited: %w", err)
	}
	return videos, nil
}

// computeAnalytics derives channel statistics from a slice of videos.
// Pure function — no DB interaction, easy to unit test.
func computeAnalytics(channelID int64, videos []ytDlpVideo) *database.ChannelAnalytics {
	if len(videos) == 0 {
		return &database.ChannelAnalytics{
			ChannelID:     channelID,
			FetchedAt:     time.Now().Unix(),
			TopTitleWords: "[]",
			TopTags:       "[]",
			SuccessTitles: "[]",
			RawTitles:     "[]",
		}
	}

	// Word frequency from titles
	wordFreq := map[string]int{}
	var rawTitles []string
	for _, v := range videos {
		rawTitles = append(rawTitles, v.Title)
		for _, word := range strings.Fields(strings.ToLower(v.Title)) {
			word = strings.Trim(word, ".,!?:;\"'()[]{}") // strip punctuation
			if len(word) >= 3 && !ptStopwords[word] {
				wordFreq[word]++
			}
		}
	}

	// Tag frequency
	tagFreq := map[string]int{}
	for _, v := range videos {
		for _, tag := range v.Tags {
			if tag != "" {
				tagFreq[strings.ToLower(tag)]++
			}
		}
	}

	// Success titles (top 10 by view_count)
	sorted := make([]ytDlpVideo, len(videos))
	copy(sorted, videos)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ViewCount > sorted[j].ViewCount })
	var successTitles []string
	for i, v := range sorted {
		if i >= 10 {
			break
		}
		successTitles = append(successTitles, v.Title)
	}

	// avg views
	totalViews := 0
	for _, v := range videos {
		totalViews += v.ViewCount
	}
	avgViews := totalViews / len(videos)

	// top 20 words
	topWords := topNWords(wordFreq, 20)
	// top 15 tags
	topTags := topNTags(tagFreq, 15)

	marshal := func(v any) string {
		b, _ := json.Marshal(v)
		return string(b)
	}

	return &database.ChannelAnalytics{
		ChannelID:     channelID,
		FetchedAt:     time.Now().Unix(),
		VideoCount:    len(videos),
		AvgViews:      avgViews,
		TopTitleWords: marshal(topWords),
		TopTags:       marshal(topTags),
		SuccessTitles: marshal(successTitles),
		RawTitles:     marshal(rawTitles),
	}
}

func topNWords(freq map[string]int, n int) []WordCount {
	wcs := make([]WordCount, 0, len(freq))
	for w, c := range freq {
		wcs = append(wcs, WordCount{Word: w, Count: c})
	}
	sort.Slice(wcs, func(i, j int) bool {
		if wcs[i].Count != wcs[j].Count {
			return wcs[i].Count > wcs[j].Count
		}
		return wcs[i].Word < wcs[j].Word
	})
	if len(wcs) > n {
		wcs = wcs[:n]
	}
	return wcs
}

func topNTags(freq map[string]int, n int) []TagCount {
	tcs := make([]TagCount, 0, len(freq))
	for t, c := range freq {
		tcs = append(tcs, TagCount{Tag: t, Count: c})
	}
	sort.Slice(tcs, func(i, j int) bool {
		if tcs[i].Count != tcs[j].Count {
			return tcs[i].Count > tcs[j].Count
		}
		return tcs[i].Tag < tcs[j].Tag
	})
	if len(tcs) > n {
		tcs = tcs[:n]
	}
	return tcs
}
