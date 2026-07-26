package api

import "testing"

func TestPlayoffBracketOptions(t *testing.T) {
	// groupSizes → ожидаемые тройки (advance, runners_up, qualifiers).
	type opt struct{ advance, runnersUp, qualifiers int }
	sizes := func(n, size int) []int {
		s := make([]int, n)
		for i := range s {
			s[i] = size
		}
		return s
	}
	cases := []struct {
		name       string
		groupSizes []int
		want       []opt
	}{
		// 2 группы по 17 — только чистые варианты (добор из 2 групп всегда
		// эквивалентен advance+1 и потому не предлагается).
		{"2x17", sizes(2, 17), []opt{{1, 0, 2}, {2, 0, 4}, {4, 0, 8}, {8, 0, 16}, {16, 0, 32}}},
		// 2 группы по 8 — до 16 команд.
		{"2x8", sizes(2, 8), []opt{{1, 0, 2}, {2, 0, 4}, {4, 0, 8}, {8, 0, 16}}},
		// 4 группы по 5 — чистые 1→4, 2→8, 4→16; добор 3×4=12→16 требует 4 ≥ numGroups → нет.
		{"4x5", sizes(4, 5), []opt{{1, 0, 4}, {2, 0, 8}, {4, 0, 16}}},
		// 3 группы по 4 (боевой кейс): чистых нет, добор в стиле Евро:
		// топ-1 + 1 лучший второй → 4; топ-2 + 2 лучших третьих → 8.
		{"3x4", sizes(3, 4), []opt{{1, 1, 4}, {2, 2, 8}}},
		// 1 группа из 20 — чистые степени двойки; добора нет (extra ≥ numGroups).
		{"1x20", sizes(1, 20), []opt{{2, 0, 2}, {4, 0, 4}, {8, 0, 8}, {16, 0, 16}}},
		// Группы разного размера 4/4/3: advance ограничен минимальной группой,
		// добор при advance=2 берёт третьих только из групп размером >2 (все три).
		{"4/4/3", []int{4, 4, 3}, []opt{{1, 1, 4}, {2, 2, 8}}},
	}
	for _, c := range cases {
		got := playoffBracketOptions(c.groupSizes)
		if len(got) != len(c.want) {
			t.Errorf("%s: получено %d вариантов, want %d (%v)", c.name, len(got), len(c.want), got)
			continue
		}
		for i, w := range c.want {
			a, _ := got[i]["advance"].(int)
			r, _ := got[i]["runners_up"].(int)
			q, _ := got[i]["qualifiers"].(int)
			if a != w.advance || r != w.runnersUp || q != w.qualifiers {
				t.Errorf("%s вариант %d = (adv=%d, ru=%d, q=%d), want (adv=%d, ru=%d, q=%d)",
					c.name, i, a, r, q, w.advance, w.runnersUp, w.qualifiers)
			}
			// qualifiers должно быть степенью двойки.
			if q < 2 || q&(q-1) != 0 {
				t.Errorf("%s вариант %d: qualifiers=%d не степень двойки", c.name, i, q)
			}
			// advance + добор согласованы: a×групп + ru == q.
			if a*len(c.groupSizes)+r != q {
				t.Errorf("%s вариант %d: %d×%d+%d != %d", c.name, i, a, len(c.groupSizes), r, q)
			}
		}
	}
}
