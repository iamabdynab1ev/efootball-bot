package engine

import (
	"reflect"
	"sort"
	"testing"
)

func TestSeedOrder(t *testing.T) {
	tests := []struct {
		size int
		want []int
	}{
		{1, []int{1}},
		{2, []int{1, 2}},
		{4, []int{1, 4, 2, 3}},
		{8, []int{1, 8, 4, 5, 2, 7, 3, 6}},
		{16, []int{1, 16, 8, 9, 4, 13, 5, 12, 2, 15, 7, 10, 3, 14, 6, 11}},
	}
	for _, tt := range tests {
		got := SeedOrder(tt.size)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SeedOrder(%d) = %v, want %v", tt.size, got, tt.want)
		}
	}
}

// SeedOrder должен быть перестановкой 1..size без пропусков и дублей.
func TestSeedOrderIsPermutation(t *testing.T) {
	for _, size := range []int{2, 4, 8, 16, 32} {
		got := SeedOrder(size)
		if len(got) != size {
			t.Fatalf("SeedOrder(%d) len=%d", size, len(got))
		}
		seen := make([]bool, size+1)
		for _, s := range got {
			if s < 1 || s > size || seen[s] {
				t.Fatalf("SeedOrder(%d) = %v не перестановка (повтор/выход за диапазон: %d)", size, got, s)
			}
			seen[s] = true
		}
	}
}

// Сильнейшие сиды должны быть в разных половинах — 1 и 2 встречаются
// только в финале.
func TestSeedOrderTopSeedsSplit(t *testing.T) {
	for _, size := range []int{4, 8, 16, 32} {
		order := SeedOrder(size)
		half := size / 2
		var pos1, pos2 int
		for i, s := range order {
			if s == 1 {
				pos1 = i
			}
			if s == 2 {
				pos2 = i
			}
		}
		if (pos1 < half) == (pos2 < half) {
			t.Errorf("size=%d: сиды 1 и 2 в одной половине (pos1=%d pos2=%d)", size, pos1, pos2)
		}
	}
}

func TestFirstRoundPowerOfTwo(t *testing.T) {
	// 8 участников — степень двойки, bye быть не должно, 4 матча.
	matches, byes := FirstRound(8)
	if len(byes) != 0 {
		t.Errorf("FirstRound(8): ожидалось 0 bye, получено %d", len(byes))
	}
	if len(matches) != 4 {
		t.Fatalf("FirstRound(8): ожидалось 4 матча, получено %d", len(matches))
	}
	// Пара сильнейшего: сид 1 vs сид 8.
	if matches[0].HomeSeed != 1 || matches[0].AwaySeed != 8 {
		t.Errorf("FirstRound(8) slot1 = %dv%d, want 1v8", matches[0].HomeSeed, matches[0].AwaySeed)
	}
}

func TestFirstRoundWithByes(t *testing.T) {
	// 6 участников → сетка 8, два bye (сиды 1 и 2), два реальных матча.
	matches, byes := FirstRound(6)
	if len(byes) != 2 {
		t.Fatalf("FirstRound(6): ожидалось 2 bye, получено %d (%+v)", len(byes), byes)
	}
	if len(matches) != 2 {
		t.Fatalf("FirstRound(6): ожидалось 2 матча, получено %d (%+v)", len(matches), matches)
	}

	// Bye достаются топ-сидам.
	byeSeeds := []int{byes[0].Seed, byes[1].Seed}
	sort.Ints(byeSeeds)
	if !reflect.DeepEqual(byeSeeds, []int{1, 2}) {
		t.Errorf("bye-сиды = %v, want [1 2]", byeSeeds)
	}

	// Каждый реальный матч — между существующими сидами (≤6).
	for _, m := range matches {
		if m.HomeSeed > 6 || m.AwaySeed > 6 {
			t.Errorf("матч содержит несуществующий сид: %+v", m)
		}
	}

	// Bye и матчи не должны указывать на один и тот же next-слот одной стороной.
	type place struct {
		slot int
		home bool
	}
	seen := map[place]bool{}
	for _, b := range byes {
		p := place{b.NextSlot, b.IsHome}
		if seen[p] {
			t.Errorf("две сущности претендуют на один next-слот/сторону: %+v", p)
		}
		seen[p] = true
	}
}

// Каждый из topK участников должен попасть в сетку ровно один раз —
// либо как игрок реального матча, либо как bye. Никто не «теряется».
func TestFirstRoundNoPlayerDropped(t *testing.T) {
	for topK := 2; topK <= 33; topK++ {
		matches, byes := FirstRound(topK)
		seen := make([]bool, topK+1)
		mark := func(seed int) {
			if seed < 1 || seed > topK {
				t.Fatalf("topK=%d: сид вне диапазона: %d", topK, seed)
			}
			if seen[seed] {
				t.Fatalf("topK=%d: сид %d встречается дважды", topK, seed)
			}
			seen[seed] = true
		}
		for _, m := range matches {
			mark(m.HomeSeed)
			mark(m.AwaySeed)
		}
		for _, b := range byes {
			mark(b.Seed)
		}
		for s := 1; s <= topK; s++ {
			if !seen[s] {
				t.Errorf("topK=%d: сид %d потерян (не в матчах и не в bye)", topK, s)
			}
		}
		// Кол-во bye = size - topK.
		wantByes := nextPow2(topK) - topK
		if len(byes) != wantByes {
			t.Errorf("topK=%d: bye=%d, want %d", topK, len(byes), wantByes)
		}
	}
}

func TestNextPow2(t *testing.T) {
	cases := map[int]int{0: 1, 1: 1, 2: 2, 3: 4, 5: 8, 6: 8, 8: 8, 9: 16, 16: 16, 17: 32}
	for in, want := range cases {
		if got := nextPow2(in); got != want {
			t.Errorf("nextPow2(%d) = %d, want %d", in, got, want)
		}
	}
}
