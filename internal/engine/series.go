package engine

// SeriesNeeded — число побед, необходимое для победы в серии best-of-N
// (ceil(N/2)). Для N=1 → 1, N=3 → 2, N=5 → 3.
func SeriesNeeded(bestOf int) int {
	if bestOf < 1 {
		bestOf = 1
	}
	return bestOf/2 + 1
}

// SeriesDecided сообщает, завершена ли серия best-of-N при счёте homeWins:awayWins,
// и кто победитель (homeWinner=true — победила домашняя сторона).
func SeriesDecided(homeWins, awayWins, bestOf int) (decided, homeWinner bool) {
	need := SeriesNeeded(bestOf)
	if homeWins >= need {
		return true, true
	}
	if awayWins >= need {
		return true, false
	}
	return false, false
}
