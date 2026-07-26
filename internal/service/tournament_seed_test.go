package service

import (
	"strconv"
	"efootball-bot/internal/models"
	"sort"
	"testing"
)

// 24 участника (например 12 из каждой из 2 групп) → сетка на 32 с 8 bye.
// Проверяем, что bye РАСПРЕДЕЛЕНЫ по сетке (в каждом слоте r16 — топ-сид против
// победителя отбора), а не свалены в одну половину.
func TestBuildSeededBracket_24WithDistributedByes(t *testing.T) {
	participants := make([]int64, 24)
	for i := range participants {
		participants[i] = int64(i + 1) // сид i+1 → userID i+1
	}
	slots, matches := buildSeededBracket(7, participants)

	byStage := map[string][]*models.BracketSlot{}
	for _, s := range slots {
		byStage[s.Stage] = append(byStage[s.Stage], s)
	}

	// Первая стадия — r32 (8 реальных матчей), последующая — r16 (8 слотов).
	if got := len(byStage[models.StageR32]); got != 8 {
		t.Errorf("r32 слотов = %d, want 8", got)
	}
	if got := len(byStage[models.StageR16]); got != 8 {
		t.Errorf("r16 слотов = %d, want 8", got)
	}
	if got := len(matches); got != 8 {
		t.Errorf("стартовых матчей = %d, want 8 (только реальные, без bye)", got)
	}

	// Ключевая проверка: в КАЖДОМ слоте r16 один участник уже известен (bye-сид),
	// второй пуст (ждёт победителя отбора). Это и есть распределённый посев.
	byesInR16 := 0
	for _, s := range byStage[models.StageR16] {
		oneKnown := (s.HomeUserID != nil) != (s.AwayUserID != nil)
		if !oneKnown {
			t.Errorf("r16 слот %d: ожидался ровно один известный участник (bye), got home=%v away=%v",
				s.Slot, s.HomeUserID, s.AwayUserID)
		}
		if s.HomeUserID != nil || s.AwayUserID != nil {
			byesInR16++
		}
	}
	if byesInR16 != 8 {
		t.Errorf("bye в r16 = %d, want 8 (по одному в каждом слоте — распределены)", byesInR16)
	}

	// Никто не потерян и нет дублей: 8 матчей×2 + 8 bye = 24 участника.
	seen := map[int64]bool{}
	add := func(id *int64) {
		if id == nil {
			return
		}
		if seen[*id] {
			t.Errorf("участник %d встречается дважды", *id)
		}
		seen[*id] = true
	}
	for _, s := range byStage[models.StageR32] {
		add(s.HomeUserID)
		add(s.AwayUserID)
	}
	for _, s := range byStage[models.StageR16] {
		add(s.HomeUserID)
		add(s.AwayUserID)
	}
	if len(seen) != 24 {
		t.Errorf("размещено участников = %d, want 24 (никто не отсечён)", len(seen))
	}

	// Топ-сиды (1 и 2) должны получить bye (не играть первый раунд).
	for _, s := range byStage[models.StageR32] {
		for _, id := range []*int64{s.HomeUserID, s.AwayUserID} {
			if id != nil && (*id == 1 || *id == 2) {
				t.Errorf("сид %d не должен играть отбор (должен получить bye)", *id)
			}
		}
	}
}

// Степень двойки (16) — bye нет, всё в первой стадии.
func TestBuildSeededBracket_16NoByes(t *testing.T) {
	participants := make([]int64, 16)
	for i := range participants {
		participants[i] = int64(i + 1)
	}
	slots, matches := buildSeededBracket(1, participants)
	r16 := 0
	for _, s := range slots {
		if s.Stage == models.StageR16 {
			r16++
		}
	}
	if r16 != 8 || len(matches) != 8 {
		t.Errorf("16 игроков: r16=%d matches=%d, want 8/8 (без bye)", r16, len(matches))
	}
}

// rankedQualifiers чередует группы по позиции: победители групп — топ-сиды.
func TestRankedQualifiers_InterleavesByPosition(t *testing.T) {
	gm := map[string][]*models.LeagueMember{
		"A": {{UserID: 101}, {UserID: 102}, {UserID: 103}},
		"B": {{UserID: 201}, {UserID: 202}, {UserID: 203}},
	}
	got := rankedQualifiers(gm, []string{"A", "B"}, 3)
	want := []int64{101, 201, 102, 202, 103, 203} // 1A,1B,2A,2B,3A,3B
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("позиция %d = %d, want %d", i, got[i], want[i])
		}
	}
	_ = sort.Ints
}

// Боевой кейс «Дмитровские🎯»: 9 участников (3 группы × 3 из группы) → сетка на 16
// с 7 bye. Пары второй стадии, где ОБА игрока прошли по bye, должны получить
// матчи сразу при генерации — иначе плей-офф зависает навсегда.
func TestBuildSeededBracket_ByePairsGetMatches(t *testing.T) {
	participants := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9}
	slots, matches := buildSeededBracket(1, participants)

	// Каждый слот, у которого известны обе стороны, обязан иметь матч.
	matchBySlot := map[string]bool{}
	for _, m := range matches {
		if m.BracketSlot != nil {
			matchBySlot[m.Stage+"#"+itoa(*m.BracketSlot)] = true
		}
	}
	for _, s := range slots {
		if s.HomeUserID != nil && s.AwayUserID != nil && !matchBySlot[s.Stage+"#"+itoa(s.Slot)] {
			t.Errorf("слот %s#%d: обе стороны известны (%d vs %d), но матч не создан",
				s.Stage, s.Slot, *s.HomeUserID, *s.AwayUserID)
		}
	}

	// 9 игроков: 1 матч первой стадии (r16) + 3 полных пары четвертьфинала.
	r16Matches, qfMatches := 0, 0
	for _, m := range matches {
		switch m.Stage {
		case models.StageR16:
			r16Matches++
		case models.StageQF:
			qfMatches++
		}
	}
	if r16Matches != 1 || qfMatches != 3 {
		t.Errorf("матчи: r16=%d qf=%d, want 1/3", r16Matches, qfMatches)
	}

	// Все участники расставлены, никто не потерян и не задвоен.
	seen := map[int64]int{}
	for _, s := range slots {
		if s.Stage != models.StageR16 && s.Stage != models.StageQF {
			continue
		}
		if s.HomeUserID != nil {
			seen[*s.HomeUserID]++
		}
		if s.AwayUserID != nil {
			seen[*s.AwayUserID]++
		}
	}
	for _, p := range participants {
		if seen[p] != 1 {
			t.Errorf("игрок %d встречается в стартовых стадиях %d раз, want 1", p, seen[p])
		}
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
