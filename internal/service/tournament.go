package service

import (
	"context"
	"efootball-bot/internal/repository"
)

// TournamentConfig holds auto-calculated parameters for a given league size and format.
type TournamentConfig struct {
	NumGroups    int
	GroupAdvance int
	BestRunnersUp int
	NumDivisions int
	NumRounds    int
}

// Calculate derives tournament parameters from the number of members and format string.
func Calculate(n int, roundsType string) TournamentConfig {
	cfg := TournamentConfig{}
	switch roundsType {
	case "groups", "groups_playoff":
		switch {
		case n <= 8:
			cfg.NumGroups, cfg.GroupAdvance, cfg.BestRunnersUp = 2, 2, 0
		case n <= 16:
			cfg.NumGroups, cfg.GroupAdvance, cfg.BestRunnersUp = 4, 2, 0
		default:
			cfg.NumGroups, cfg.GroupAdvance, cfg.BestRunnersUp = 8, 2, 0
		}
	case "swiss":
		cfg.NumRounds = NumRoundsForSwiss(n)
	case "nations_league":
		switch {
		case n <= 8:
			cfg.NumDivisions = 2
		case n <= 16:
			cfg.NumDivisions = 4
		default:
			cfg.NumDivisions = 8
		}
	}
	return cfg
}

// NumRoundsForSwiss returns the number of rounds for a Swiss-system tournament.
func NumRoundsForSwiss(n int) int {
	r := 0
	for p := 1; p < n; p *= 2 {
		r++
	}
	if r < 3 {
		return 3
	}
	return r
}

// GroupStageService handles group-stage schedule generation.
type GroupStageService struct {
	matchRepo  repository.MatchRepository
	leagueRepo repository.LeagueRepository
}

func NewGroupStageService(mr repository.MatchRepository, lr repository.LeagueRepository) *GroupStageService {
	return &GroupStageService{matchRepo: mr, leagueRepo: lr}
}

func (s *GroupStageService) GenerateGroupStage(ctx context.Context, leagueID int64, numGroups, groupAdvance int) error {
	return nil
}

func (s *GroupStageService) GeneratePlayoffFromGroups(ctx context.Context, leagueID int64, groupAdvance, bestRunnersUp int, bracketRepo repository.BracketRepository) error {
	return nil
}

// CupService handles cup (knockout) bracket generation.
type CupService struct {
	matchRepo   repository.MatchRepository
	leagueRepo  repository.LeagueRepository
	bracketRepo repository.BracketRepository
}

func NewCupService(mr repository.MatchRepository, lr repository.LeagueRepository, br repository.BracketRepository) *CupService {
	return &CupService{matchRepo: mr, leagueRepo: lr, bracketRepo: br}
}

func (s *CupService) GenerateCup(ctx context.Context, leagueID int64) error {
	return nil
}

// SwissService handles Swiss-system round generation.
type SwissService struct {
	matchRepo   repository.MatchRepository
	leagueRepo  repository.LeagueRepository
	bracketRepo repository.BracketRepository
}

func NewSwissService(mr repository.MatchRepository, lr repository.LeagueRepository, br repository.BracketRepository) *SwissService {
	return &SwissService{matchRepo: mr, leagueRepo: lr, bracketRepo: br}
}

func (s *SwissService) GenerateFirstRound(ctx context.Context, leagueID int64) error {
	return nil
}

func (s *SwissService) GenerateNextRound(ctx context.Context, leagueID int64, maxRounds int) error {
	return nil
}

// NationsLeagueService handles Nations League format generation.
type NationsLeagueService struct {
	matchRepo   repository.MatchRepository
	leagueRepo  repository.LeagueRepository
	bracketRepo repository.BracketRepository
}

func NewNationsLeagueService(mr repository.MatchRepository, lr repository.LeagueRepository, br repository.BracketRepository) *NationsLeagueService {
	return &NationsLeagueService{matchRepo: mr, leagueRepo: lr, bracketRepo: br}
}

func (s *NationsLeagueService) GenerateNationsLeague(ctx context.Context, leagueID int64, numDivisions int) error {
	return nil
}

func (s *NationsLeagueService) GenerateFinalFour(ctx context.Context, leagueID int64) error {
	return nil
}
