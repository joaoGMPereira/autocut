package processor

import (
	"testing"
)

func TestComputeSegments(t *testing.T) {
	tests := []struct {
		name      string
		totalSec  float64
		segSec    float64
		wantCount int
		wantFirst Segment
		wantLast  Segment
	}{
		{"exact 3", 1800, 600, 3, Segment{0, 600}, Segment{1200, 1800}},
		{"merge tail", 1320, 600, 2, Segment{0, 600}, Segment{600, 1320}},
		{"separate tail", 1080, 600, 2, Segment{0, 600}, Segment{600, 1080}},
		{"shorter than seg", 300, 600, 1, Segment{0, 300}, Segment{0, 300}},
		{"zero total", 0, 600, 0, Segment{}, Segment{}},
		{"boundary 50pct", 2100, 600, 4, Segment{0, 600}, Segment{1800, 2100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeSegments(tt.totalSec, tt.segSec)
			if len(got) != tt.wantCount {
				t.Fatalf("got %d segments, want %d", len(got), tt.wantCount)
			}
			if tt.wantCount == 0 {
				return
			}
			if got[0] != tt.wantFirst {
				t.Errorf("first = %+v, want %+v", got[0], tt.wantFirst)
			}
			if got[len(got)-1] != tt.wantLast {
				t.Errorf("last = %+v, want %+v", got[len(got)-1], tt.wantLast)
			}
		})
	}
}
