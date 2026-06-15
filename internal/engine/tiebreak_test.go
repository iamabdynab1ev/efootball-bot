package engine

import (
	"reflect"
	"testing"
)

func TestStandingLess_Points(t *testing.T) {
	a := Standing{UserID: 1, Points: 9}
	b := Standing{UserID: 2, Points: 6}
	if !StandingLess(a, b) {
		t.Error("больше очков должно ставить выше")
	}
	if StandingLess(b, a) {
		t.Error("меньше очков не может быть выше")
	}
}

func TestStandingLess_GoalDiff(t *testing.T) {
	// Равные очки → решает разница мячей.
	a := Standing{UserID: 1, Points: 6, GoalsFor: 10, GoalsAgainst: 2} // gd +8
	b := Standing{UserID: 2, Points: 6, GoalsFor: 5, GoalsAgainst: 3}  // gd +2
	if !StandingLess(a, b) {
		t.Error("при равных очках выше тот, у кого лучше разница")
	}
}

func TestStandingLess_GoalsFor(t *testing.T) {
	// Равные очки и разница → решает число забитых.
	a := Standing{UserID: 1, Points: 6, GoalsFor: 10, GoalsAgainst: 5} // gd +5, gf 10
	b := Standing{UserID: 2, Points: 6, GoalsFor: 7, GoalsAgainst: 2}  // gd +5, gf 7
	if !StandingLess(a, b) {
		t.Error("при равных очках и разнице выше тот, кто больше забил")
	}
}

func TestStandingLess_FullTie(t *testing.T) {
	a := Standing{UserID: 1, Points: 6, GoalsFor: 7, GoalsAgainst: 2}
	b := Standing{UserID: 2, Points: 6, GoalsFor: 7, GoalsAgainst: 2}
	if StandingLess(a, b) || StandingLess(b, a) {
		t.Error("полное равенство базовых критериев → ни один не выше (нужен H2H)")
	}
}

func TestSortStandings(t *testing.T) {
	s := []Standing{
		{UserID: 3, Points: 3, GoalsFor: 4, GoalsAgainst: 4},
		{UserID: 1, Points: 9, GoalsFor: 10, GoalsAgainst: 1},
		{UserID: 2, Points: 6, GoalsFor: 8, GoalsAgainst: 5},
	}
	SortStandings(s)
	gotOrder := []int64{s[0].UserID, s[1].UserID, s[2].UserID}
	if !reflect.DeepEqual(gotOrder, []int64{1, 2, 3}) {
		t.Errorf("порядок = %v, want [1 2 3]", gotOrder)
	}
}

func TestRankH2H_DecidesByHeadToHead(t *testing.T) {
	// Двое равны в общей таблице; в очной встрече победил игрок 2.
	ids := []int64{1, 2}
	matches := []H2HResult{
		{HomeID: 1, AwayID: 2, HomeGoals: 0, AwayGoals: 1},
	}
	got := RankH2H(ids, matches)
	if !reflect.DeepEqual(got, []int64{2, 1}) {
		t.Errorf("RankH2H = %v, want [2 1] (победитель очной встречи выше)", got)
	}
}

func TestRankH2H_ThreeWayByGoalDiff(t *testing.T) {
	// Мини-турнир из трёх; у всех по 3 H2H-очка → решает H2H-разница.
	ids := []int64{1, 2, 3}
	matches := []H2HResult{
		{HomeID: 1, AwayID: 2, HomeGoals: 3, AwayGoals: 0}, // 1 бьёт 2 (+3)
		{HomeID: 2, AwayID: 3, HomeGoals: 2, AwayGoals: 0}, // 2 бьёт 3 (+2)
		{HomeID: 3, AwayID: 1, HomeGoals: 1, AwayGoals: 0}, // 3 бьёт 1 (+1)
	}
	// Все по 3 очка. Разница: 1 = +3-1=+2; 2 = -3+2=-1; 3 = -2+1=-1.
	// → 1 первый; 2 и 3 равны по очкам и разнице → стабильный порядок (2,3).
	got := RankH2H(ids, matches)
	if got[0] != 1 {
		t.Errorf("RankH2H первый = %d, want 1 (лучшая H2H-разница)", got[0])
	}
}

func TestRankH2H_IgnoresOutsideMatches(t *testing.T) {
	// Матч с участником вне группы не должен влиять на ранжирование.
	ids := []int64{1, 2}
	matches := []H2HResult{
		{HomeID: 1, AwayID: 99, HomeGoals: 5, AwayGoals: 0}, // 99 не в группе
		{HomeID: 2, AwayID: 1, HomeGoals: 1, AwayGoals: 0},  // 2 бьёт 1
	}
	got := RankH2H(ids, matches)
	if !reflect.DeepEqual(got, []int64{2, 1}) {
		t.Errorf("RankH2H = %v, want [2 1] (внешний матч игнорируется)", got)
	}
}

func TestRankH2H_Stable(t *testing.T) {
	// Нет очных матчей → исходный порядок сохраняется.
	ids := []int64{5, 3, 9}
	got := RankH2H(ids, nil)
	if !reflect.DeepEqual(got, []int64{5, 3, 9}) {
		t.Errorf("RankH2H без матчей = %v, want исходный [5 3 9]", got)
	}
}
