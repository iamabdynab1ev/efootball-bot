package engine

import "sort"

// Standing — минимальные данные для базовых тай-брейков по турнирной таблице.
type Standing struct {
	UserID       int64
	Points       int
	GoalsFor     int
	GoalsAgainst int
}

// GoalDiff — разница мячей.
func (s Standing) GoalDiff() int { return s.GoalsFor - s.GoalsAgainst }

// StandingLess сообщает, идёт ли a СТРОГО выше b по базовым тай-брейкам:
// очки → разница мячей → забитые мячи (всё по убыванию). Равенство по всем трём
// означает, что нужен следующий критерий (личные встречи) — здесь вернётся false
// в обе стороны.
func StandingLess(a, b Standing) bool {
	if a.Points != b.Points {
		return a.Points > b.Points
	}
	if a.GoalDiff() != b.GoalDiff() {
		return a.GoalDiff() > b.GoalDiff()
	}
	return a.GoalsFor > b.GoalsFor
}

// SortStandings сортирует таблицу по базовым тай-брейкам (стабильно).
func SortStandings(s []Standing) {
	sort.SliceStable(s, func(i, j int) bool { return StandingLess(s[i], s[j]) })
}

// H2HResult — один сыгранный матч между участниками тай-группы.
type H2HResult struct {
	HomeID, AwayID       int64
	HomeGoals, AwayGoals int
}

// RankH2H упорядочивает участников тай-группы по очной мини-таблице:
// H2H-очки (победа 3 / ничья 1) по убыванию, затем H2H-разница мячей по
// убыванию. Сортировка стабильна — участники с одинаковой H2H-статистикой
// сохраняют исходный порядок. Матчи с не-этими участниками игнорируются.
func RankH2H(userIDs []int64, matches []H2HResult) []int64 {
	type stat struct {
		pts, gd int
	}
	in := make(map[int64]bool, len(userIDs))
	stats := make(map[int64]*stat, len(userIDs))
	for _, id := range userIDs {
		in[id] = true
		stats[id] = &stat{}
	}

	for _, m := range matches {
		if !in[m.HomeID] || !in[m.AwayID] {
			continue // матч вне тай-группы — не влияет на очный зачёт
		}
		stats[m.HomeID].gd += m.HomeGoals - m.AwayGoals
		stats[m.AwayID].gd += m.AwayGoals - m.HomeGoals
		switch {
		case m.HomeGoals > m.AwayGoals:
			stats[m.HomeID].pts += 3
		case m.HomeGoals < m.AwayGoals:
			stats[m.AwayID].pts += 3
		default:
			stats[m.HomeID].pts++
			stats[m.AwayID].pts++
		}
	}

	out := make([]int64, len(userIDs))
	copy(out, userIDs)
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := stats[out[i]], stats[out[j]]
		if si.pts != sj.pts {
			return si.pts > sj.pts
		}
		return si.gd > sj.gd
	})
	return out
}
