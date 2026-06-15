package engine

import "testing"

func TestSeriesNeeded(t *testing.T) {
	cases := map[int]int{0: 1, 1: 1, 2: 2, 3: 2, 4: 3, 5: 3, 7: 4}
	for bestOf, want := range cases {
		if got := SeriesNeeded(bestOf); got != want {
			t.Errorf("SeriesNeeded(%d) = %d, want %d", bestOf, got, want)
		}
	}
}

func TestSeriesDecided(t *testing.T) {
	tests := []struct {
		name                 string
		h, a, bestOf         int
		wantDecided, wantHome bool
	}{
		{"bo1 home wins", 1, 0, 1, true, true},
		{"bo1 away wins", 0, 1, 1, true, false},
		{"bo3 ongoing 1-0", 1, 0, 3, false, false},
		{"bo3 ongoing 1-1", 1, 1, 3, false, false},
		{"bo3 home clinches 2-0", 2, 0, 3, true, true},
		{"bo3 home clinches 2-1", 2, 1, 3, true, true},
		{"bo3 away clinches 1-2", 1, 2, 3, true, false},
		{"bo5 ongoing 2-2", 2, 2, 5, false, false},
		{"bo5 home clinches 3-2", 3, 2, 5, true, true},
		{"bo5 away clinches 0-3", 0, 3, 5, true, false},
	}
	for _, tt := range tests {
		decided, home := SeriesDecided(tt.h, tt.a, tt.bestOf)
		if decided != tt.wantDecided || (decided && home != tt.wantHome) {
			t.Errorf("%s: SeriesDecided(%d,%d,%d) = (%v,%v), want (%v,%v)",
				tt.name, tt.h, tt.a, tt.bestOf, decided, home, tt.wantDecided, tt.wantHome)
		}
	}
}
