package api

import "testing"

func TestPlayoffBracketOptions(t *testing.T) {
	// (numGroups, minSize) → ожидаемые пары (advance, qualifiers).
	type pair struct{ advance, qualifiers int }
	cases := []struct {
		numGroups, minSize int
		want               []pair
	}{
		// 2 группы по 17 — варианты до 1/16 финала (32 команды).
		{2, 17, []pair{{1, 2}, {2, 4}, {4, 8}, {8, 16}, {16, 32}}},
		// 2 группы по 8 — до 1/8 финала (16 команд), 16 из группы недоступно.
		{2, 8, []pair{{1, 2}, {2, 4}, {4, 8}, {8, 16}}},
		// 4 группы по 5 — advance×4: 1→4, 2→8, 4→16 (3 не степень двойки).
		{4, 5, []pair{{1, 4}, {2, 8}, {4, 16}}},
		// 3 группы — advance×3 никогда не степень двойки → нет ровных вариантов.
		{3, 10, nil},
		// 1 группа из 20 — степени двойки до 16 (на 32 нужно ≥32 в группе).
		{1, 20, []pair{{2, 2}, {4, 4}, {8, 8}, {16, 16}}},
	}
	for _, c := range cases {
		got := playoffBracketOptions(c.numGroups, c.minSize)
		if len(got) != len(c.want) {
			t.Errorf("(%d,%d): получено %d вариантов, want %d (%v)", c.numGroups, c.minSize, len(got), len(c.want), got)
			continue
		}
		for i, w := range c.want {
			a, _ := got[i]["advance"].(int)
			q, _ := got[i]["qualifiers"].(int)
			if a != w.advance || q != w.qualifiers {
				t.Errorf("(%d,%d) вариант %d = (adv=%d, q=%d), want (adv=%d, q=%d)",
					c.numGroups, c.minSize, i, a, q, w.advance, w.qualifiers)
			}
			// qualifiers должно быть степенью двойки.
			if q < 2 || q&(q-1) != 0 {
				t.Errorf("(%d,%d) вариант %d: qualifiers=%d не степень двойки", c.numGroups, c.minSize, i, q)
			}
		}
	}
}
