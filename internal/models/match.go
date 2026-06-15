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

// Stages for playoff bracket
const (
	StageLeague = "league"
	StageR32    = "r32"
	StageR16    = "r16"
	StageQF     = "qf"
	StageSF     = "sf"
	StageFinal  = "final"

	// Double-elimination — стадии-маркеры (раунд/позиция хранятся в de_nodes).
	StageDEWinners = "de_w"  // верхняя сетка
	StageDELosers  = "de_l"  // нижняя сетка
	StageDEGrand   = "de_gf" // гранд-финал (+ reset)
)

var StageOrder = []string{StageR32, StageR16, StageQF, StageSF, StageFinal}

var StageLabel = map[string]string{
	StageR32:   "1/16 финала",
	StageR16:   "1/8 финала",
	StageQF:    "Четвертьфинал",
	StageSF:    "Полуфинал",
	StageFinal: "Финал",
}

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
	Stage        string      `db:"stage"`
	BracketSlot  *int        `db:"bracket_slot"`

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

// IsKnockoutStage returns true for any on-bracket stage (single-elim r32..final
// и стадии двойной элиминации) — такие матчи не идут в турнирную таблицу.
func IsKnockoutStage(stage string) bool {
	switch stage {
	case StageR32, StageR16, StageQF, StageSF, StageFinal,
		StageDEWinners, StageDELosers, StageDEGrand:
		return true
	}
	return false
}

// IsGroupStage returns true for round-robin group/division/Swiss-round stages
// (group letters "A".."H", "div_X", "swiss_rN") — i.e. neither knockout nor
// the legacy unstaged league ("").
func IsGroupStage(stage string) bool {
	if stage == "" || stage == StageLeague || IsKnockoutStage(stage) {
		return false
	}
	return true
}

// IsTableStage reports whether a confirmed match in this stage should update
// league_members standings (points, W/D/L, goals, position).
//
// Knockout-stage results never affect group/division standings — except for
// the pure "cup" format, where the knockout bracket IS the whole competition
// and the standings table doubles as a simple results log.
func IsTableStage(roundsType, stage string) bool {
	if !IsKnockoutStage(stage) {
		return true // "", "league", group letters, div_X, swiss_rN
	}
	return GetFormat(roundsType) == FormatCup
}
