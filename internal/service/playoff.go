package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"errors"
)

type PlayoffService struct {
	leagueRepo  repository.LeagueRepository
	bracketRepo repository.BracketRepository
}

func NewPlayoffService(lr repository.LeagueRepository, br repository.BracketRepository) *PlayoffService {
	return &PlayoffService{leagueRepo: lr, bracketRepo: br}
}

// GeneratePlayoff creates a seeded single-elimination bracket from league standings.
// topK — how many teams advance (will be rounded down to previous power of 2).
func (s *PlayoffService) GeneratePlayoff(ctx context.Context, leagueID int64, topK int) error {
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

	// First-stage matches
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

	// Слоты и матчи создаются атомарно под advisory-lock лиги: параллельный
	// второй вызов получит ErrBracketExists вместо дублей.
	return s.bracketRepo.GenerateBracket(ctx, leagueID, allSlots, matches)
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

	// Определяем победителя. Ничья в плей-офф недопустима — матч должен быть оспорен.
	var winnerID int64
	if *match.HomeGoals > *match.AwayGoals {
		winnerID = match.HomeUserID
	} else if *match.AwayGoals > *match.HomeGoals {
		winnerID = match.AwayUserID
	} else {
		// Ничья в плей-офф: возвращаем ошибку — admin должен разрешить спор с чётким счётом.
		return errors.New("draw in knockout match is not allowed: admin must set a decisive score via dispute resolution")
	}

	// Запись победителя, посев следующего слота и создание матча следующей
	// стадии выполняются одной транзакцией под advisory-lock лиги: два
	// одновременных подтверждения, ведущих в один слот, сериализуются.
	p := repository.AdvanceParams{
		LeagueID:  match.LeagueID,
		Stage:     match.Stage,
		Slot:      *match.BracketSlot,
		WinnerID:  winnerID,
		MatchID:   match.ID,
		NextStage: models.NextStage(match.Stage), // пусто после финала
		NewRound:  match.Round + 1,
	}
	if p.NextStage != "" {
		// Winner fills slot ceil(N/2) of the next stage.
		// Odd-numbered slots → home; even-numbered → away.
		p.NextSlot = (*match.BracketSlot + 1) / 2
		p.IsHome = *match.BracketSlot%2 == 1
	}
	_, err := s.bracketRepo.AdvanceSlot(ctx, p)
	return err
}
