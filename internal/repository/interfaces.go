package repository

import (
	"context"
	"efootball-bot/internal/models"
	"time"
)

type UserRepository interface {
	// Существующие методы (бот)
	Create(ctx context.Context, telegramID int64, displayName string, username *string) (*models.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	UpdateDisplayName(ctx context.Context, id int64, name string) error
	UpdateRating(ctx context.Context, userID int64, newRating int) error
	// GetRatingHistory — последние точки ELO в хронологическом порядке (график).
	GetRatingHistory(ctx context.Context, userID int64, limit int) ([]models.RatingPoint, error)
	UpdateTeamPower(ctx context.Context, userID int64, tp int) error
	// TouchLastSeen отмечает момент активности пользователя (для «был(а) в сети»).
	TouchLastSeen(ctx context.Context, userID int64) error
	GetTopScorers(ctx context.Context, leagueID int64) ([]*models.LeagueMember, error)
	GetTopScorersAllLeagues(ctx context.Context) ([]*LeagueWithScorers, error)
	UpdateRank(ctx context.Context, userID int64, rank string) error
	RecalculateAllRanks(ctx context.Context) error
	ResetAllRatings(ctx context.Context) error
	UpdateLanguage(ctx context.Context, userID int64, lang string) error
	UpdateFavoriteClub(ctx context.Context, userID int64, clubID string) error

	// Новые методы (web)
	UpsertByGoogle(ctx context.Context, googleID, email, displayName string) (*models.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*models.User, error)
	GetAllByRating(ctx context.Context, limit int) ([]*models.User, error)
	GenerateLinkCode(ctx context.Context, userID int64) (string, error)
	LinkTelegramByCode(ctx context.Context, code string, telegramID int64, username *string) (*models.User, error)
	// UnlinkTelegram отвязывает Telegram от аккаунта (telegram_id = NULL).
	UnlinkTelegram(ctx context.Context, userID int64) error
	GetGlobalStats(ctx context.Context, userID int64) (*models.GlobalStats, error)
	// GetAllTelegramIDs — все привязанные telegram_id (для админ-рассылки).
	GetAllTelegramIDs(ctx context.Context) ([]int64, error)
	// DeleteUser полностью удаляет пользователя и все его связи. Возвращает id
	// лиг, затронутых удалением (для пересчёта таблиц).
	DeleteUser(ctx context.Context, userID int64) ([]int64, error)
}

type LeagueRepository interface {
	GetActiveLeagues(ctx context.Context) ([]*models.League, error)
	GetAllLeagues(ctx context.Context) ([]*models.League, error)
	GetAllPendingMembers(ctx context.Context) ([]*models.LeagueMember, error)
	GetByID(ctx context.Context, id int64) (*models.League, error)
	GetByName(ctx context.Context, name string) (*models.League, error)
	GetOrCreateActiveSeason(ctx context.Context) (*models.Season, error)
	GetSeasonByID(ctx context.Context, id int64) (*models.Season, error)
	GetLatestClosedSeason(ctx context.Context) (*models.Season, error)
	ListLeaguesBySeason(ctx context.Context, seasonID int64) ([]*models.League, error)
	SeasonAggregates(ctx context.Context, seasonID int64) ([]*SeasonAggregate, error)
	SeasonTotals(ctx context.Context, seasonID int64) (matches, goals int, err error)
	EloDeltasSince(ctx context.Context, since time.Time) (map[int64]int, error)
	// CloseSeason закрывает сезон и открывает следующий (одна транзакция).
	CloseSeason(ctx context.Context, seasonID int64, nextName string) (*models.Season, error)
	CreateLeague(ctx context.Context, seasonID int64, name string, deadline *time.Time, roundsType string, numGroups, groupAdvance, bestRunnersUp int16) (*models.League, error)
	SetLeagueStatus(ctx context.Context, leagueID int64, status string) error
	UpdateLeague(ctx context.Context, id int64, name string, deadline *time.Time) error
	ArchiveLeague(ctx context.Context, leagueID int64) error
	DeleteLeague(ctx context.Context, leagueID int64) error
	GetUserLeagues(ctx context.Context, userID int64) ([]*models.LeagueMember, error)

	AddMember(ctx context.Context, leagueID, userID int64) error
	ApproveMember(ctx context.Context, leagueID, userID int64) error
	RejectMember(ctx context.Context, leagueID, userID int64) error
	GetPendingMembers(ctx context.Context, leagueID int64) ([]*models.LeagueMember, error)
	ApplyMatchResultStats(ctx context.Context, leagueID, homeUserID, awayUserID int64, hg, ag int16) error
	GetMembers(ctx context.Context, leagueID int64) ([]*models.LeagueMember, error)
	IsMember(ctx context.Context, leagueID, userID int64) (bool, error)
	GetMemberStats(ctx context.Context, leagueID, userID int64) (*models.LeagueMember, error)
	RecalculateTable(ctx context.Context, leagueID int64) error
	SetMemberPosition(ctx context.Context, memberID int64, position int16) error
	// ResetMemberStats обнуляет статистику участников лиги (для полного пересчёта).
	ResetMemberStats(ctx context.Context, leagueID int64) error
	RemoveMember(ctx context.Context, leagueID, userID int64) error
	// SetMemberGroups назначает группы пачке участников одним UPDATE
	// (userIDs[i] → groups[i]).
	SetMemberGroups(ctx context.Context, leagueID int64, userIDs []int64, groups []string) error
	GetGroupRunnersUp(ctx context.Context, leagueID int64, groupAdvance int) ([]*models.LeagueMember, error)
	GetMembersByGroup(ctx context.Context, leagueID int64, groupName string) ([]*models.LeagueMember, error)
	GetLeagueGroups(ctx context.Context, leagueID int64) ([]string, error)
	SetMemberDivision(ctx context.Context, leagueID, userID int64, division string) error
	GetMembersByDivision(ctx context.Context, leagueID int64, division string) ([]*models.LeagueMember, error)
	GetLeagueDivisions(ctx context.Context, leagueID int64) ([]string, error)
	SetCurrentRound(ctx context.Context, leagueID int64, round int16) error
	SetLeagueGroupConfig(ctx context.Context, leagueID int64, numGroups, groupAdvance int) error
	SetLeagueBestOf(ctx context.Context, leagueID int64, bestOf int) error
}

type MatchRepository interface {
	CreateBatch(ctx context.Context, matches []*models.Match) error
	HasMatches(ctx context.Context, leagueID int64) (bool, error)
	CountUnconfirmedLeagueMatches(ctx context.Context, leagueID int64) (int, error)
	GetByID(ctx context.Context, id int64) (*models.Match, error)
	GetPendingForUser(ctx context.Context, userID int64) ([]*models.Match, error)
	GetUserSchedule(ctx context.Context, userID, leagueID int64) ([]*models.Match, error)
	GetScheduleForLeague(ctx context.Context, leagueID int64, round int16) ([]*models.Match, error)
	GetAllForLeague(ctx context.Context, leagueID int64) ([]*models.Match, error)
	GetMatchesByStage(ctx context.Context, leagueID int64, stage string) ([]*models.Match, error)
	GetConfirmedMatchesBetween(ctx context.Context, leagueID int64, userIDs []int64, stage string) ([]*models.Match, error)
	GetUserMatchHistory(ctx context.Context, userID int64, limit, offset int, leagueID int64) ([]*models.Match, error)
	CareerStats(ctx context.Context, userID int64) (played int, goals int, err error)
	// GetConfirmedBetweenUsers — все подтверждённые матчи между двумя игроками
	// (по всем лигам), для личных встреч (H2H). Свежие сначала.
	GetConfirmedBetweenUsers(ctx context.Context, userA, userB int64, limit int) ([]*models.Match, error)
	GetAllDisputed(ctx context.Context) ([]*models.Match, error)
	GetAllLeagueForm(ctx context.Context, leagueID int64) (map[int64][]string, error)

	ClaimResult(ctx context.Context, matchID int64, homeGoals, awayGoals int16) error
	Confirm(ctx context.Context, matchID int64) (bool, error)
	Dispute(ctx context.Context, matchID int64, homeClaimed, awayClaimed int16) error
	AdminResolve(ctx context.Context, matchID int64, homeGoals, awayGoals int16, adminID int64, note string) error
	// SetMatchScore — админ-перезапись счёта любого матча (включая подтверждённый).
	SetMatchScore(ctx context.Context, matchID int64, homeGoals, awayGoals int16) error
	// ClearMatchScore — админ-отмена результата (матч → scheduled, счёт очищен).
	ClearMatchScore(ctx context.Context, matchID int64) error

	// Best-of-X серии.
	// RecordSeriesGame увеличивает счёт серии на победу одной из сторон и
	// возвращает новый счёт.
	RecordSeriesGame(ctx context.Context, matchID int64, homeWon bool) (homeWins, awayWins int16, err error)
	// ReopenForNextGame возвращает матч в 'scheduled' и очищает заявку/счёт
	// для следующей игры серии (счёт серии сохраняется).
	ReopenForNextGame(ctx context.Context, matchID int64) error
	// SetSeriesAggregate записывает итог серии в home_goals/away_goals, чтобы
	// логика продвижения (по голам) определила победителя серии.
	SetSeriesAggregate(ctx context.Context, matchID int64, homeWins, awayWins int16) error
}

// AdvanceParams описывает один подтверждённый результат плей-офф:
// победитель (Stage, Slot) переходит в NextSlot стадии NextStage.
type AdvanceParams struct {
	LeagueID  int64
	Stage     string
	Slot      int
	WinnerID  int64
	MatchID   int64
	NextStage string // пусто, если сыгран финал
	NextSlot  int
	IsHome    bool
	NewRound  int16
}

type BracketRepository interface {
	// GenerateBracket атомарно создаёт все слоты сетки и стартовые матчи
	// в одной транзакции под advisory-lock лиги и линкует match_id в слоты.
	// Возвращает ErrBracketExists, если сетка уже сгенерирована.
	GenerateBracket(ctx context.Context, leagueID int64, slots []*models.BracketSlot, matches []*models.Match) error
	// AdvanceSlot атомарно записывает результат матча плей-офф и, когда оба
	// участника следующего слота известны, создаёт матч следующей стадии.
	// Возвращает созданный матч или nil, если слот ещё не готов.
	AdvanceSlot(ctx context.Context, p AdvanceParams) (*models.Match, error)
	// SeedSlotSide — посадить игрока в сторону слота (матч за 3-е место);
	// матч создаётся, когда обе стороны известны.
	SeedSlotSide(ctx context.Context, leagueID int64, stage string, slot int, isHome bool, userID int64, newRound int16) (*models.Match, error)
	GetAllSlots(ctx context.Context, leagueID int64) ([]*models.BracketSlot, error)
	HasBracket(ctx context.Context, leagueID int64) (bool, error)
}

// DoubleElimRepository — персистентность сетки двойной элиминации.
type DoubleElimRepository interface {
	HasDoubleElim(ctx context.Context, leagueID int64) (bool, error)
	// GenerateDoubleElim атомарно (advisory-lock лиги) создаёт все узлы графа и
	// стартовые матчи (узлы, у которых оба участника уже известны). Возвращает
	// ErrBracketExists, если сетка уже сгенерирована.
	GenerateDoubleElim(ctx context.Context, leagueID int64, nodes []*models.DENode, bestOf int16) error
	// AdvanceDoubleElim записывает победителя матча, маршрутизирует победителя и
	// проигравшего по графу и создаёт готовые матчи следующих узлов — всё в одной
	// транзакции под advisory-lock. Возвращает id чемпиона (когда определён) и
	// список вновь созданных матчей.
	AdvanceDoubleElim(ctx context.Context, leagueID, matchID, winnerID, loserID int64, bestOf int16) (champion *int64, created []*models.Match, err error)
	GetDoubleElimNodes(ctx context.Context, leagueID int64) ([]*models.DENode, error)
}

type AchievementRepository interface {
	GetAll(ctx context.Context) ([]*models.Achievement, error)
	GetUserAchievements(ctx context.Context, userID int64) ([]*models.UserAchievement, error)
	HasAchievement(ctx context.Context, userID int64, code string, leagueID *int64) (bool, error)
	// Award выдаёт достижение; inserted=true — только если оно реально новое
	// (для уведомления без дублей при повторных проверках).
	Award(ctx context.Context, userID int64, code string, leagueID *int64) (inserted bool, err error)
	// GetByCode — достижение по коду (для текста уведомления).
	GetByCode(ctx context.Context, code string) (*models.Achievement, error)
	// GetUnclaimed — достижения, ещё не «забранные» игроком (claimed_at IS NULL).
	GetUnclaimed(ctx context.Context, userID int64) ([]*models.UserAchievement, error)
	// ClaimAll — помечает все неполученные достижения игрока полученными.
	ClaimAll(ctx context.Context, userID int64) (int64, error)
}

type DeadlineRepository interface {
	// SetDeadline — upsert дедлайна: round>0 (stage="") для тура, stage!=""
	// (round=0) для стадии плей-офф. Перенос срока в будущее сбрасывает флаги
	// напоминаний и processed_at — автоматика отработает заново.
	SetDeadline(ctx context.Context, leagueID int64, round int, stage string, deadline time.Time) error
	GetDeadlines(ctx context.Context, leagueID int64) ([]*models.RoundDeadline, error)
	GetPendingReminders(ctx context.Context, now time.Time) ([]*models.RoundDeadline, error)
	MarkReminderSent(ctx context.Context, id int64, is24h bool) error
	DeleteDeadline(ctx context.Context, leagueID int64, round int, stage string) error
	// DueUnprocessed — истёкшие необработанные дедлайны активных лиг.
	DueUnprocessed(ctx context.Context, now time.Time) ([]*models.RoundDeadline, error)
	MarkProcessed(ctx context.Context, id int64) error
}

type AwardRepository interface {
	// CreateAward — идемпотентно; inserted=true только для нового трофея.
	CreateAward(ctx context.Context, seasonID, leagueID int64, awardType string, userID int64, value int) (inserted bool, err error)
	// CreateSeasonAward — сезонная номинация без привязки к лиге (league_id NULL).
	CreateSeasonAward(ctx context.Context, seasonID int64, awardType string, userID int64, value int) (inserted bool, err error)
	GetBySeason(ctx context.Context, seasonID int64) ([]*models.SeasonAward, error)
	GetAll(ctx context.Context) ([]*models.SeasonAward, error)
	GetByUser(ctx context.Context, userID int64) ([]*models.SeasonAward, error)
	// GetUnclaimedByUser — трофеи игрока, ещё не «забранные» (claimed_at IS NULL).
	GetUnclaimedByUser(ctx context.Context, userID int64) ([]*models.SeasonAward, error)
	// ClaimAll — помечает все неполученные трофеи игрока полученными.
	ClaimAll(ctx context.Context, userID int64) (int64, error)
}

// StatEntry — универсальная строка для всех стат-рейтингов.
type StatEntry struct {
	UserID       int64
	DisplayName  string
	FavoriteClub *string
	Rank         string
	Rating       int
	Played       int
	Wins         int
	Draws        int
	Losses       int
	GoalsFor     int
	GoalsAgainst int
	TeamPower    int
	WinRate      float64 // процент побед (0–100)
	AvgGoals     float64 // средних голов за матч
	Streak       int     // текущая серия побед
}

type StatsRepository interface {
	GetWinRate(ctx context.Context, minGames int) ([]*StatEntry, error)
	GetStreaks(ctx context.Context) ([]*StatEntry, error)
	GetAvgGoals(ctx context.Context, minGames int) ([]*StatEntry, error)
	GetTeamPower(ctx context.Context) ([]*StatEntry, error)
	GetActivity(ctx context.Context) ([]*StatEntry, error)
}
