package processor

import (
	"strings"
	"testing"
	"time"
)

// MockExecutor captures every Run invocation for assertion.
type MockExecutor struct {
	calls []mockCall
	// ReturnErr, if non-nil, is returned by every Run call.
	ReturnErr error
	// ReturnOut is the []byte returned by every Run call.
	ReturnOut []byte
}

type mockCall struct {
	name string
	args []string
}

func (m *MockExecutor) Run(name string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, mockCall{name: name, args: args})
	if m.ReturnErr != nil {
		return m.ReturnOut, m.ReturnErr
	}
	return m.ReturnOut, nil
}

func (m *MockExecutor) lastArgs() []string {
	if len(m.calls) == 0 {
		return nil
	}
	return m.calls[len(m.calls)-1].args
}

func (m *MockExecutor) allArgs() []string {
	var all []string
	for _, c := range m.calls {
		all = append(all, c.args...)
	}
	return all
}

func containsSeq(args []string, seq ...string) bool {
	for i := 0; i <= len(args)-len(seq); i++ {
		match := true
		for j, s := range seq {
			if args[i+j] != s {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func containsArg(args []string, arg string) bool {
	for _, a := range args {
		if a == arg {
			return true
		}
	}
	return false
}

// ---- CutClip ----

func TestCutClip_ArgsContainSSAndTO(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	start := 10 * time.Second
	end := 30 * time.Second

	// Use /dev/null as output to avoid real file creation.
	_ = p.CutClip("/input.mp4", start, end, "/dev/null/output.mp4")

	args := mock.lastArgs()
	if args == nil {
		t.Fatal("no ffmpeg call recorded")
	}

	if !containsArg(args, "-ss") {
		t.Error("expected -ss flag")
	}
	if !containsArg(args, "-to") {
		t.Error("expected -to flag")
	}
	// Verify start timestamp appears after -ss
	if !containsSeq(args, "-ss", formatDuration(start)) {
		t.Errorf("expected -ss %s, got args: %v", formatDuration(start), args)
	}
	// Verify end timestamp appears after -to
	if !containsSeq(args, "-to", formatDuration(end)) {
		t.Errorf("expected -to %s, got args: %v", formatDuration(end), args)
	}
}

func TestCutClip_ArgsContainCCopy(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	_ = p.CutClip("/input.mp4", 0, 5*time.Second, "/dev/null/out.mp4")

	args := mock.lastArgs()
	if !containsSeq(args, "-c", "copy") {
		t.Errorf("expected -c copy, got args: %v", args)
	}
}

func TestCutClip_TimestampFormat(t *testing.T) {
	d := 1*time.Hour + 2*time.Minute + 3*time.Second + 500*time.Millisecond
	got := formatDuration(d)
	want := "01:02:03.500"
	if got != want {
		t.Errorf("formatDuration(%v) = %q, want %q", d, got, want)
	}
}

// ---- ExtractFrame ----

func TestExtractFrame_ArgsContainFramesV1(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	_ = p.ExtractFrame("/input.mp4", 5*time.Second, "/dev/null/frame.jpg")

	args := mock.lastArgs()
	if args == nil {
		t.Fatal("no ffmpeg call recorded")
	}
	if !containsSeq(args, "-frames:v", "1") {
		t.Errorf("expected -frames:v 1, got args: %v", args)
	}
}

func TestExtractFrame_ContainsSS(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	at := 15 * time.Second
	_ = p.ExtractFrame("/input.mp4", at, "/dev/null/frame.jpg")

	args := mock.lastArgs()
	if !containsSeq(args, "-ss", formatDuration(at)) {
		t.Errorf("expected -ss %s, got: %v", formatDuration(at), args)
	}
}

// ---- MergeClips ----

func TestMergeClips_ArgsContainConcat(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	_ = p.MergeClips([]string{"/a.mp4", "/b.mp4"}, "/dev/null/out.mp4")

	// MergeClips calls ffmpeg once with -f concat
	var foundConcat bool
	for _, c := range mock.calls {
		if containsSeq(c.args, "-f", "concat") {
			foundConcat = true
			break
		}
	}
	if !foundConcat {
		t.Errorf("expected -f concat in ffmpeg args, calls: %+v", mock.calls)
	}
}

func TestMergeClips_ArgsSafe0(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	_ = p.MergeClips([]string{"/a.mp4"}, "/dev/null/out.mp4")

	var allArgs []string
	for _, c := range mock.calls {
		allArgs = append(allArgs, c.args...)
	}
	if !containsSeq(allArgs, "-safe", "0") {
		t.Errorf("expected -safe 0, got: %v", allArgs)
	}
}

func TestMergeClips_EmptyInputsReturnsError(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	err := p.MergeClips(nil, "/dev/null/out.mp4")
	if err == nil {
		t.Error("expected error for empty input list")
	}
}

// ---- ApplySpeedZones ----

func TestApplySpeedZones_FilterContainsAtempo(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	zones := []SpeedZone{
		{Start: 0, End: 10 * time.Second, Speed: 1.5},
	}
	_ = p.ApplySpeedZones("/input.mp4", zones, "/dev/null/out.mp4")

	args := mock.lastArgs()
	if args == nil {
		t.Fatal("no ffmpeg call recorded")
	}

	// Find -filter_complex value
	var fc string
	for i, a := range args {
		if a == "-filter_complex" && i+1 < len(args) {
			fc = args[i+1]
			break
		}
	}
	if fc == "" {
		t.Fatalf("no -filter_complex in args: %v", args)
	}
	if !strings.Contains(fc, "atempo") {
		t.Errorf("expected atempo in filter_complex, got: %s", fc)
	}
}

func TestApplySpeedZones_FilterContainsSetpts(t *testing.T) {
	mock := &MockExecutor{}
	p := newFFmpegProcessorWithExecutor("ffmpeg", mock)

	zones := []SpeedZone{
		{Start: 0, End: 5 * time.Second, Speed: 2.0},
	}
	_ = p.ApplySpeedZones("/input.mp4", zones, "/dev/null/out.mp4")

	args := mock.lastArgs()
	var fc string
	for i, a := range args {
		if a == "-filter_complex" && i+1 < len(args) {
			fc = args[i+1]
			break
		}
	}
	if !strings.Contains(fc, "setpts") {
		t.Errorf("expected setpts in filter_complex, got: %s", fc)
	}
}

// ---- buildAtempoChain ----

func TestBuildAtempoChain_Simple(t *testing.T) {
	tests := []struct {
		speed float64
		want  string
	}{
		{1.5, "atempo=1.5000"},
		{2.0, "atempo=2.0000"},
	}
	for _, tt := range tests {
		got := buildAtempoChain(tt.speed)
		if got != tt.want {
			t.Errorf("buildAtempoChain(%.1f) = %q, want %q", tt.speed, got, tt.want)
		}
	}
}

func TestBuildAtempoChain_ChainedFor3x(t *testing.T) {
	got := buildAtempoChain(3.0)
	// 3x requires chaining: atempo=2.0,atempo=1.5
	if !strings.Contains(got, "atempo=2.0") {
		t.Errorf("expected atempo=2.0 in chain for 3x, got: %s", got)
	}
	parts := strings.Split(got, ",")
	if len(parts) < 2 {
		t.Errorf("expected chained atempo for 3x, got single filter: %s", got)
	}
}
