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
	LeagueDraft        LeagueStatus = "draft"        // создан, ещё не открыт
	LeagueRegistration LeagueStatus = "registration" // приём заявок
	LeagueActive       LeagueStatus = "active"       // идут матчи
	LeagueFinished     LeagueStatus = "finished"     // завершён
	LeagueArchived     LeagueStatus = "archived"     // в архиве
)

// Форматы турниров
const (
	FormatLeague        = "league"         // Лига (круговой)
	FormatCup           = "cup"            // Кубок (вылет)
	FormatGroupsPlayoff = "groups_playoff" // Группы + Плей-офф
	FormatSwiss         = "swiss"          // Швейцарская система
	FormatNationsLeague = "nations_league" // Лига Наций
	FormatHybrid        = "hybrid"         // Гибридный: авто-группы + плей-офф
	FormatDoubleElim    = "double_elim"    // Двойная элиминация (WB/LB/grand final)
)

// GetFormat возвращает унифицированный формат по rounds_type.
func GetFormat(roundsType string) string {
	switch roundsType {
	case "cup":
		return FormatCup
	case "groups", "groups_playoff", "hybrid":
		return FormatGroupsPlayoff
	case "swiss":
		return FormatSwiss
	case "nations_league":
		return FormatNationsLeague
	case "double_elim":
		return FormatDoubleElim
	default:
		return FormatLeague // single | double → лига
	}
}

type League struct {
	ID                   int64        `db:"id"`
	SeasonID             int64        `db:"season_id"`
	Name                 string       `db:"name"`
	Country              *string      `db:"country"`
	Level                int16        `db:"level"`
	MaxPlayers           int16        `db:"max_players"`
	RoundsType           string       `db:"rounds_type"` // "single"|"double"|"groups"|"cup"|"swiss"|"nations_league"
	NumGroups            int16        `db:"num_groups"`
	GroupAdvance         int16        `db:"group_advance"`
	BestRunnersUp        int16        `db:"best_runners_up"` // лучших 2-х мест в плей-офф (FIFA режим)
	BestOf               int16        `db:"best_of"`         // длина серии для матчей на выбывание (1 — одиночный)
	CurrentRound         int16        `db:"current_round"`
	Status               LeagueStatus `db:"status"`
	RegistrationDeadline *time.Time   `db:"registration_deadline"`
	CreatedAt            time.Time    `db:"created_at"`
	UpdatedAt            time.Time    `db:"updated_at"`

	// MemberCount — реальное число одобренных участников (вычисляется в запросе,
	// не колонка). Для отображения «X игроков» вместо вместимости max_players.
	MemberCount int16 `db:"-"`
}

func (l *League) Format() string { return GetFormat(l.RoundsType) }

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
	GroupName    string       `db:"group_name"` // "A","B","C","D" — только для groups формата
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
