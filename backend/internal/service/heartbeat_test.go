package service

import (
	"testing"
	"time"
)

func TestCapPlayMsBoundsByElapsedPlusTolerance(t *testing.T) {
	cases := []struct {
		name     string
		playMs   int
		elapsed  time.Duration
		hasPrior bool
		want     int
	}{
		{"normal cadence passes", 10000, 10 * time.Second, true, 10000},
		{"delayed tick within tolerance passes", 20000, 10 * time.Second, true, 20000},
		{"instant huge claim clamps to tolerance", 3600000, 0, true, 15000},
		{"instant 35s clamps to tolerance", 35000, time.Second, true, 15000 + 1000},
		{"slow client keeps elapsed allowance", 60000, 50 * time.Second, true, 60000},
		{"over elapsed clamps", 90000, 50 * time.Second, true, 50000 + 15000},
		{"negative elapsed treated as zero", 20000, -5 * time.Second, true, 15000},
		{"first heartbeat capped at tolerance", 12000, 0, false, 12000},
		{"first heartbeat huge claim clamps", 30000, 0, false, 15000},
		{"zero play stays zero", 0, 10 * time.Second, true, 0},
	}
	for _, tc := range cases {
		if got := capPlayMs(tc.playMs, tc.elapsed, tc.hasPrior); got != tc.want {
			t.Errorf("%s: capPlayMs(%d, %v, %v) = %d, want %d",
				tc.name, tc.playMs, tc.elapsed, tc.hasPrior, got, tc.want)
		}
	}
}

func TestCapPlayMsE2ECumulativeDeltasPass(t *testing.T) {
	// The T08 happy path posts 12s deltas back-to-back; each must survive
	// the cap so three of them still cross the 30s threshold.
	prior := 0
	for i := 0; i < 3; i++ {
		prior += capPlayMs(12000, 500*time.Millisecond, i > 0)
	}
	if prior != 36000 {
		t.Fatalf("cumulative capped deltas = %d, want 36000", prior)
	}
	if prior < ValidStreamMs {
		t.Fatalf("capped deltas %d do not cross %d", prior, ValidStreamMs)
	}
}
