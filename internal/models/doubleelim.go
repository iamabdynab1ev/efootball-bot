package models

// DENode — узел сетки двойной элиминации (строка de_nodes), обогащённый
// JOIN-полями для отображения. Источники участников кодируются строками:
//
//	"seed:N" — начальный сид N
//	"win:K"  — победитель узла node_key=K
//	"lose:K" — проигравший узла node_key=K
type DENode struct {
	ID           int64
	LeagueID     int64
	NodeKey      int
	Bracket      string // StageDEWinners | StageDELosers | StageDEGrand
	Round        int
	Ord          int
	IsReset      bool
	HomeUserID   *int64
	AwayUserID   *int64
	HomeSrc      *string
	AwaySrc      *string
	MatchID      *int64
	WinnerUserID *int64

	// JOIN для отображения
	HomeName    string
	AwayName    string
	WinnerName  string
	HomeClub    string
	AwayClub    string
	HomeGoals   *int16
	AwayGoals   *int16
	MatchStatus string
}
