package models

import "time"

type BracketSlot struct {
	ID           int64
	LeagueID     int64
	Stage        string
	Slot         int
	HomeUserID   *int64
	AwayUserID   *int64
	MatchID      *int64
	WinnerUserID *int64
	CreatedAt    time.Time

	// Joined for display
	HomeName    string
	AwayName    string
	WinnerName  string
	HomeClub    string
	AwayClub    string
	WinnerClub  string
	HomeGoals   *int16
	AwayGoals   *int16
	MatchStatus string
}

// NextStage returns the next stage after the given one, empty string if final.
func NextStage(stage string) string {
	for i, s := range StageOrder {
		if s == stage && i+1 < len(StageOrder) {
			return StageOrder[i+1]
		}
	}
	return ""
}

// FirstStageForSize returns the bracket stage name for N teams.
func FirstStageForSize(n int) string {
	switch {
	case n > 16:
		return StageR32
	case n > 8:
		return StageR16
	case n > 4:
		return StageQF
	case n > 2:
		return StageSF
	default:
		return StageFinal
	}
}

// PrevPowerOf2 returns the largest power of 2 ≤ n.
func PrevPowerOf2(n int) int {
	if n <= 0 {
		return 1
	}
	p := 1
	for p*2 <= n {
		p *= 2
	}
	return p
}
