package processor

import (
	"fmt"
	"testing"
	"time"
)

// mockFFmpegProcessor tracks which methods were called on it.
// Implements ffmpegInterface.
type mockFFmpegProcessor struct {
	removeSilenceCalled    bool
	removeSilenceThreshold float64
	applySpeedZonesCalled  bool
	applySpeedZonesCount   int
	cutClipCalled          bool

	failRemoveSilence   bool
	failApplySpeedZones bool
}

func (m *mockFFmpegProcessor) CutClip(input string, start, end time.Duration, output string) error {
	m.cutClipCalled = true
	if output == "" {
		return fmt.Errorf("mock: empty output")
	}
	return nil
}

func (m *mockFFmpegProcessor) RemoveSilence(input string, threshold float64, minDuration time.Duration, output string) error {
	m.removeSilenceCalled = true
	m.removeSilenceThreshold = threshold
	if m.failRemoveSilence {
		return fmt.Errorf("mock: RemoveSilence forced failure")
	}
	// Create a minimal stub output so probeDuration does not blow up during tests.
	// (probeDuration is not called in unit tests since ffprobeBin is "false".)
	return nil
}

func (m *mockFFmpegProcessor) ApplySpeedZones(input string, zones []SpeedZone, output string) error {
	m.applySpeedZonesCalled = true
	m.applySpeedZonesCount = len(zones)
	if m.failApplySpeedZones {
		return fmt.Errorf("mock: ApplySpeedZones forced failure")
	}
	return nil
}

// newTestOptimizer builds an optimizer that uses a mock ffmpeg and a no-op ffprobe bin.
// We use "false" as ffprobeBin so probeDuration always returns an error (0 duration),
// which is acceptable because tests do not assert on duration values.
func newTestOptimizer(mock *mockFFmpegProcessor) *VideoOptimizerProcessor {
	return newVideoOptimizerProcessorWithMock(mock, "false")
}

// ---- Tests ----

func TestOptimize_CallsRemoveSilence_WhenThresholdNonZero(t *testing.T) {
	mock := &mockFFmpegProcessor{}
	opt := newTestOptimizer(mock)

	cfg := OptimizerConfig{
		SilenceThreshold:   -30.0,
		MinSilenceDuration: 500 * time.Millisecond,
	}

	// probeDuration will fail (ffprobeBin="false"), so Optimize returns an error,
	// but RemoveSilence must still have been called before that.
	_, _ = opt.Optimize("/input.mp4", cfg, "/tmp/out.mp4")

	if !mock.removeSilenceCalled {
		t.Error("expected RemoveSilence to be called when SilenceThreshold != 0")
	}
	if mock.removeSilenceThreshold != -30.0 {
		t.Errorf("expected threshold -30.0, got %f", mock.removeSilenceThreshold)
	}
}

func TestOptimize_DoesNotCallRemoveSilence_WhenThresholdZero(t *testing.T) {
	mock := &mockFFmpegProcessor{}
	opt := newTestOptimizer(mock)

	cfg := OptimizerConfig{
		SilenceThreshold: 0, // disabled
	}
	_, _ = opt.Optimize("/input.mp4", cfg, "/tmp/out.mp4")

	if mock.removeSilenceCalled {
		t.Error("expected RemoveSilence NOT to be called when SilenceThreshold == 0")
	}
}

func TestOptimize_CallsApplySpeedZones_WhenZonesProvided(t *testing.T) {
	mock := &mockFFmpegProcessor{}
	opt := newTestOptimizer(mock)

	cfg := OptimizerConfig{
		SpeedZones: []SpeedZone{
			{Start: 0, End: 10 * time.Second, Speed: 1.5},
			{Start: 20 * time.Second, End: 30 * time.Second, Speed: 2.0},
		},
	}
	_, _ = opt.Optimize("/input.mp4", cfg, "/tmp/out.mp4")

	if !mock.applySpeedZonesCalled {
		t.Error("expected ApplySpeedZones to be called when SpeedZones are provided")
	}
	if mock.applySpeedZonesCount != 2 {
		t.Errorf("expected 2 zones passed to ApplySpeedZones, got %d", mock.applySpeedZonesCount)
	}
}

func TestOptimize_DoesNotCallApplySpeedZones_WhenNoZones(t *testing.T) {
	mock := &mockFFmpegProcessor{}
	opt := newTestOptimizer(mock)

	cfg := OptimizerConfig{SpeedZones: nil}
	_, _ = opt.Optimize("/input.mp4", cfg, "/tmp/out.mp4")

	if mock.applySpeedZonesCalled {
		t.Error("expected ApplySpeedZones NOT to be called when no zones provided")
	}
}

func TestOptimize_BothStages_SilenceAndSpeed(t *testing.T) {
	mock := &mockFFmpegProcessor{}
	opt := newTestOptimizer(mock)

	cfg := OptimizerConfig{
		SilenceThreshold:   -40.0,
		MinSilenceDuration: 300 * time.Millisecond,
		SpeedZones: []SpeedZone{
			{Start: 0, End: 5 * time.Second, Speed: 1.5},
		},
	}
	_, _ = opt.Optimize("/input.mp4", cfg, "/tmp/out.mp4")

	if !mock.removeSilenceCalled {
		t.Error("expected RemoveSilence called in combined run")
	}
	if !mock.applySpeedZonesCalled {
		t.Error("expected ApplySpeedZones called in combined run")
	}
}

func TestOptimize_ReturnsError_WhenRemoveSilenceFails(t *testing.T) {
	mock := &mockFFmpegProcessor{failRemoveSilence: true}
	opt := newTestOptimizer(mock)

	cfg := OptimizerConfig{SilenceThreshold: -30.0}
	_, err := opt.Optimize("/input.mp4", cfg, "/tmp/out.mp4")
	if err == nil {
		t.Error("expected error when RemoveSilence fails")
	}
}
