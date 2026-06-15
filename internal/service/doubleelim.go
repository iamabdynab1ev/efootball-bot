package service

import (
	"context"
	"efootball-bot/internal/engine"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"errors"
	"fmt"
)

type DoubleElimService struct {
	leagueRepo repository.LeagueRepository
	deRepo     repository.DoubleElimRepository
}

func NewDoubleElimService(lr repository.LeagueRepository, de repository.DoubleElimRepository) *DoubleElimService {
	return &DoubleElimService{leagueRepo: lr, deRepo: de}
}

// ErrNotPowerOfTwo — двойная элиминация v1 требует степень двойки участников
// (bye в нижней сетке усложняют маршрутизацию проигравших — вне scope v1).
var ErrNotPowerOfTwo = errors.New("double elimination requires a power-of-two number of teams (2, 4, 8, 16, 32)")

// Generate строит сетку двойной элиминации из топ-topK участников таблицы.
func (s *DoubleElimService) Generate(ctx context.Context, leagueID int64, topK int) error {
	standings, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return err
	}
	if len(standings) < 2 {
		return errors.New("not enough players for double elimination")
	}
	if topK <= 0 || topK > len(standings) {
		topK = len(standings)
	}
	if topK > 32 {
		topK = 32
	}
	if topK < 2 || topK&(topK-1) != 0 {
		return ErrNotPowerOfTwo
	}

	participants := make([]int64, topK)
	for i := 0; i < topK; i++ {
		participants[i] = standings[i].UserID
	}

	league, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil || league == nil {
		return errors.New("league not found")
	}
	bestOf := league.BestOf
	if bestOf < 1 {
		bestOf = 1
	}

	graph, err := engine.DoubleElim(topK)
	if err != nil {
		return err
	}

	nodes := make([]*models.DENode, 0, len(graph))
	for _, g := range graph {
		dn := &models.DENode{
			NodeKey: g.ID,
			Bracket: bracketStage(g.Bracket),
			Round:   g.Round,
			Ord:     g.Order,
			IsReset: g.Reset,
		}
		dn.HomeSrc, dn.HomeUserID = encodeRef(g.A, participants)
		dn.AwaySrc, dn.AwayUserID = encodeRef(g.B, participants)
		nodes = append(nodes, dn)
	}

	return s.deRepo.GenerateDoubleElim(ctx, leagueID, nodes, bestOf)
}

// AdvanceByMatch продвигает граф после подтверждения DE-матча. Возвращает id
// чемпиона, когда он определён (победитель гранд-финала / reset).
func (s *DoubleElimService) AdvanceByMatch(ctx context.Context, match *models.Match) (*int64, []*models.Match, error) {
	if match.HomeGoals == nil || match.AwayGoals == nil {
		return nil, nil, nil
	}
	var winnerID, loserID int64
	switch {
	case *match.HomeGoals > *match.AwayGoals:
		winnerID, loserID = match.HomeUserID, match.AwayUserID
	case *match.AwayGoals > *match.HomeGoals:
		winnerID, loserID = match.AwayUserID, match.HomeUserID
	default:
		return nil, nil, errors.New("draw in double-elimination match is not allowed")
	}
	bestOf := match.BestOf
	if bestOf < 1 {
		bestOf = 1
	}
	return s.deRepo.AdvanceDoubleElim(ctx, match.LeagueID, match.ID, winnerID, loserID, bestOf)
}

func (s *DoubleElimService) Has(ctx context.Context, leagueID int64) (bool, error) {
	return s.deRepo.HasDoubleElim(ctx, leagueID)
}

// bracketStage маппит часть сетки на stage-строку матча.
func bracketStage(b engine.Bracket) string {
	switch b {
	case engine.Winners:
		return models.StageDEWinners
	case engine.Losers:
		return models.StageDELosers
	default:
		return models.StageDEGrand
	}
}

// encodeRef кодирует источник участника и, если это сид, сразу резолвит userID.
func encodeRef(ref engine.Ref, participants []int64) (*string, *int64) {
	if ref.Seed > 0 {
		src := fmt.Sprintf("seed:%d", ref.Seed)
		uid := participants[ref.Seed-1]
		return &src, &uid
	}
	verb := "win"
	if ref.Loser {
		verb = "lose"
	}
	src := fmt.Sprintf("%s:%d", verb, ref.Node)
	return &src, nil
}
