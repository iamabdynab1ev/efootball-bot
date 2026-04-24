package models

import "time"

type MatchStatus string

const (
	MatchScheduled      MatchStatus = "scheduled"
	MatchPendingConfirm MatchStatus = "pending_confirm"
	MatchDisputed       MatchStatus = "disputed"
	MatchConfirmed      MatchStatus = "confirmed"
	MatchCancelled      MatchStatus = "cancelled"
)

type Match struct {
	ID         int64 `db:"id"`
	LeagueID   int64 `db:"league_id"`
	HomeUserID int64 `db:"home_user_id"`
	AwayUserID int64 `db:"away_user_id"`
	Round      int16 `db:"round"`

	// Финальный счёт (после confirmed)
	HomeGoals *int16 `db:"home_goals"`
	AwayGoals *int16 `db:"away_goals"`

	// Что заявил хозяин (текущая заявка)
	ClaimedHome *int16 `db:"claimed_home"`
	ClaimedAway *int16 `db:"claimed_away"`

	Status       MatchStatus `db:"status"`
	DisputeCount int16       `db:"dispute_count"`

	PlayedAt  *time.Time `db:"played_at"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`

	// JOIN поля
	HomeUser *User `db:"-"`
	AwayUser *User `db:"-"`
}

// ─────────────────────────────────────────

type Dispute struct {
	ID          int64 `db:"id"`
	MatchID     int64 `db:"match_id"`
	ReportedBy  int64 `db:"reported_by"`
	HomeClaimed int16 `db:"home_claimed"`
	AwayClaimed int16 `db:"away_claimed"`

	Resolved       bool    `db:"resolved"`
	ResolvedBy     *int64  `db:"resolved_by"`
	AdminHomeGoals *int16  `db:"admin_home_goals"`
	AdminAwayGoals *int16  `db:"admin_away_goals"`
	AdminNote      *string `db:"admin_note"`

	CreatedAt  time.Time  `db:"created_at"`
	ResolvedAt *time.Time `db:"resolved_at"`
}

func (m *Match) GetHomeGoals() int16 {
	if m.HomeGoals != nil {
		return *m.HomeGoals
	}
	return 0
}

func (m *Match) GetAwayGoals() int16 {
	if m.AwayGoals != nil {
		return *m.AwayGoals
	}
	return 0
}
