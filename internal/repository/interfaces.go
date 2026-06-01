package repository

import (
	"context"
	"efootball-bot/internal/models"
)

type UserRepository interface {
	// Существующие методы (бот)
	Create(ctx context.Context, telegramID int64, displayName string, username *string) (*models.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	UpdateDisplayName(ctx context.Context, id int64, name string) error
	UpdateRating(ctx context.Context, userID int64, newRating int) error
	UpdateTeamPower(ctx context.Context, userID int64, tp int) error
	GetTopScorers(ctx context.Context, leagueID int64) ([]*models.LeagueMember, error)
	UpdateRank(ctx context.Context, userID int64, rank string) error
	RecalculateAllRanks(ctx context.Context) error
	ResetAllRatings(ctx context.Context) error
	UpdateLanguage(ctx context.Context, userID int64, lang string) error

	// Новые методы (web)
	UpsertByGoogle(ctx context.Context, googleID, email, displayName string) (*models.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*models.User, error)
	GetAllByRating(ctx context.Context, limit int) ([]*models.User, error)
	GenerateLinkCode(ctx context.Context, userID int64) (string, error)
	LinkTelegramByCode(ctx context.Context, code string, telegramID int64, username *string) (*models.User, error)
}

type LeagueRepository interface {
	GetActiveLeagues(ctx context.Context) ([]*models.League, error)
	GetAllLeagues(ctx context.Context) ([]*models.League, error)
	GetAllPendingMembers(ctx context.Context) ([]*models.LeagueMember, error)
	GetByID(ctx context.Context, id int64) (*models.League, error)
	GetByName(ctx context.Context, name string) (*models.League, error)
	GetOrCreateActiveSeason(ctx context.Context) (*models.Season, error)
	CreateLeague(ctx context.Context, seasonID int64, name string) (*models.League, error)
	SetLeagueStatus(ctx context.Context, leagueID int64, status string) error
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
	RemoveMember(ctx context.Context, leagueID, userID int64) error
}

type MatchRepository interface {
	CreateBatch(ctx context.Context, matches []*models.Match) error
	HasMatches(ctx context.Context, leagueID int64) (bool, error)
	GetByID(ctx context.Context, id int64) (*models.Match, error)
	GetPendingForUser(ctx context.Context, userID int64) ([]*models.Match, error)
	GetUserSchedule(ctx context.Context, userID, leagueID int64) ([]*models.Match, error)
	GetScheduleForLeague(ctx context.Context, leagueID int64, round int16) ([]*models.Match, error)
	GetAllForLeague(ctx context.Context, leagueID int64) ([]*models.Match, error)
	GetUserMatchHistory(ctx context.Context, userID int64, limit, offset int) ([]*models.Match, error)
	GetAllDisputed(ctx context.Context) ([]*models.Match, error)

	ClaimResult(ctx context.Context, matchID int64, homeGoals, awayGoals int16) error
	Confirm(ctx context.Context, matchID int64) (bool, error)
	Dispute(ctx context.Context, matchID int64, homeClaimed, awayClaimed int16) error
	AdminResolve(ctx context.Context, matchID int64, homeGoals, awayGoals int16, adminID int64, note string) error
}

