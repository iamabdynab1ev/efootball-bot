package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"errors"
)

type PlayoffService struct {
	matchRepo   repository.MatchRepository
	leagueRepo  repository.LeagueRepository
	bracketRepo repository.BracketRepository
}

func NewPlayoffService(mr repository.MatchRepository, lr repository.LeagueRepository, br repository.BracketRepository) *PlayoffService {
	return &PlayoffService{matchRepo: mr, leagueRepo: lr, bracketRepo: br}
}

// GeneratePlayoff creates a seeded single-elimination bracket from league standings.
// topK — how many teams advance (will be rounded down to previous power of 2).
func (s *PlayoffService) GeneratePlayoff(ctx context.Context, leagueID int64, topK int) error {
	already, err := s.bracketRepo.HasBracket(ctx, leagueID)
	if err != nil {
		return err
	}
	if already {
		return errors.New("playoff bracket already generated")
	}

	// Get standings sorted by position
	standings, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return err
	}
	if len(standings) < 2 {
		return errors.New("not enough players in standings")
	}

	// Clamp and round down to power of 2
	if topK <= 0 || topK > len(standings) {
		topK = len(standings)
	}
	topK = models.PrevPowerOf2(topK)
	if topK < 2 {
		return errors.New("need at least 2 teams for playoff")
	}

	// Take top K players by position
	participants := make([]int64, 0, topK)
	for i, m := range standings {
		if i >= topK {
			break
		}
		participants = append(participants, m.UserID)
	}

	// First stage name based on bracket size
	firstStage := models.FirstStageForSize(topK)

	// Build all bracket slots for all stages
	var allSlots []*models.BracketSlot

	// First stage: seeded pairs — 1v(N), 2v(N-1), ...
	matchesInStage := topK / 2
	for i := 0; i < matchesInStage; i++ {
		home := participants[i]
		away := participants[topK-1-i]
		allSlots = append(allSlots, &models.BracketSlot{
			LeagueID:   leagueID,
			Stage:      firstStage,
			Slot:       i + 1,
			HomeUserID: &home,
			AwayUserID: &away,
		})
	}

	// Subsequent stages: empty slots (filled as winners advance)
	stage := models.NextStage(firstStage)
	for stage != "" {
		count := matchesInStage / 2
		if count < 1 {
			count = 1
		}
		for i := 1; i <= count; i++ {
			allSlots = append(allSlots, &models.BracketSlot{
				LeagueID: leagueID,
				Stage:    stage,
				Slot:     i,
			})
		}
		matchesInStage = count
		stage = models.NextStage(stage)
	}

	// Persist bracket slots
	if err := s.bracketRepo.CreateSlots(ctx, allSlots); err != nil {
		return err
	}

	// Create first-stage matches
	var matches []*models.Match
	roundNum := int16(100) // start from 100 to distinguish from league rounds
	for _, slot := range allSlots {
		if slot.Stage != firstStage {
			continue
		}
		slotCopy := slot.Slot
		matches = append(matches, &models.Match{
			LeagueID:    leagueID,
			HomeUserID:  *slot.HomeUserID,
			AwayUserID:  *slot.AwayUserID,
			Round:       roundNum,
			Stage:       firstStage,
			BracketSlot: &slotCopy,
		})
		roundNum++
	}

	return s.matchRepo.CreateBatch(ctx, matches)
}

// AdvanceBracket is called after a knockout match is confirmed.
// It records the winner and creates the next-round match if both participants are known.
func (s *PlayoffService) AdvanceBracket(ctx context.Context, match *models.Match) error {
	if match.Stage == "" || match.Stage == models.StageLeague || match.BracketSlot == nil {
		return nil
	}
	if match.HomeGoals == nil || match.AwayGoals == nil {
		return nil
	}

	// Determine winner (no draws in knockout)
	var winnerID int64
	if *match.HomeGoals > *match.AwayGoals {
		winnerID = match.HomeUserID
	} else if *match.AwayGoals > *match.HomeGoals {
		winnerID = match.AwayUserID
	} else {
		// Draw in knockout — home team advances (admin should resolve via dispute)
		winnerID = match.HomeUserID
	}

	// Record winner in current slot
	if err := s.bracketRepo.SetWinner(ctx, match.LeagueID, match.Stage, *match.BracketSlot, winnerID, match.ID); err != nil {
		return err
	}

	// Find next stage
	nextStage := models.NextStage(match.Stage)
	if nextStage == "" {
		// Final played — tournament over
		return nil
	}

	// Winner fills slot ceil(N/2) of the next stage
	// Odd-numbered slots → home; even-numbered → away
	nextSlot := (*match.BracketSlot + 1) / 2
	isHome := *match.BracketSlot%2 == 1

	if err := s.bracketRepo.SetParticipant(ctx, match.LeagueID, nextStage, nextSlot, winnerID, isHome); err != nil {
		return err
	}

	// Check if both participants of next slot are ready → create match
	slot, err := s.bracketRepo.GetSlot(ctx, match.LeagueID, nextStage, nextSlot)
	if err != nil || slot == nil {
		return err
	}

	if slot.HomeUserID == nil || slot.AwayUserID == nil || slot.MatchID != nil {
		return nil // not ready or match already created
	}

	// Create the match
	slotNum := nextSlot
	var maxRound int16
	existing, _ := s.matchRepo.GetAllForLeague(ctx, match.LeagueID)
	for _, m := range existing {
		if m.Round > maxRound {
			maxRound = m.Round
		}
	}
	newRound := maxRound + 1

	newMatch := &models.Match{
		LeagueID:    match.LeagueID,
		HomeUserID:  *slot.HomeUserID,
		AwayUserID:  *slot.AwayUserID,
		Round:       newRound,
		Stage:       nextStage,
		BracketSlot: &slotNum,
	}
	if err := s.matchRepo.CreateBatch(ctx, []*models.Match{newMatch}); err != nil {
		return err
	}

	// Get the newly created match to link it back to the slot
	all, err := s.matchRepo.GetAllForLeague(ctx, match.LeagueID)
	if err != nil {
		return err
	}
	for _, m := range all {
		if m.Stage == nextStage && m.BracketSlot != nil && *m.BracketSlot == nextSlot {
			return s.bracketRepo.SetMatchID(ctx, match.LeagueID, nextStage, nextSlot, m.ID)
		}
	}
	return nil
}
