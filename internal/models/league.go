package models

import "time"

type SeasonStatus string

const (
	SeasonPending  SeasonStatus = "pending"
	SeasonActive   SeasonStatus = "active"
	SeasonFinished SeasonStatus = "finished"
)

type Season struct {
	ID        int64        `db:"id"`
	Name      string       `db:"name"`
	Status    SeasonStatus `db:"status"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt time.Time    `db:"updated_at"`
}

type LeagueStatus string

const (
	LeagueRegistration LeagueStatus = "registration"
	LeagueActive       LeagueStatus = "active"
	LeagueFinished     LeagueStatus = "finished"
	LeagueArchived     LeagueStatus = "archived"
)

type League struct {
	ID         int64        `db:"id"`
	SeasonID   int64        `db:"season_id"`
	Name       string       `db:"name"`
	Country    *string      `db:"country"`
	Level      int16        `db:"level"` // 1=нац, 2=ЛЧ, 3=ЛЕ
	MaxPlayers int16        `db:"max_players"`
	RoundsType string       `db:"rounds_type"` // "single" | "double"
	Status     LeagueStatus `db:"status"`
	CreatedAt  time.Time    `db:"created_at"`
	UpdatedAt  time.Time    `db:"updated_at"`
}

// ─────────────────────────────────────────

type MemberStatus string

const (
	MemberPending  MemberStatus = "pending"
	MemberApproved MemberStatus = "approved"
	MemberRejected MemberStatus = "rejected"
)

type LeagueMember struct {
	ID           int64        `db:"id"`
	LeagueID     int64        `db:"league_id"`
	UserID       int64        `db:"user_id"`
	Status       MemberStatus `db:"status"`
	Points       int16        `db:"points"`
	Wins         int16        `db:"wins"`
	Draws        int16        `db:"draws"`
	Losses       int16        `db:"losses"`
	GoalsFor     int16        `db:"goals_for"`
	GoalsAgainst int16        `db:"goals_against"`
	Position     *int16       `db:"position"`
	JoinedAt     time.Time    `db:"joined_at"`
	UpdatedAt    time.Time    `db:"updated_at"`

	User   *User   `db:"-"`
	League *League `db:"-"`
}

// GoalDiff возвращает разницу голов
func (m *LeagueMember) GoalDiff() int16 {
	return m.GoalsFor - m.GoalsAgainst
}
