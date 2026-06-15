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
// topK — сколько команд проходит в плей-офф. Если topK не степень двойки, сетка
// дополняется до ближайшей степени двойки, а топ-сиды получают bye (проход без
// игры в следующий раунд) — участники больше НЕ отсекаются.
func (s *PlayoffService) GeneratePlayoff(ctx context.Context, leagueID int64, topK int) error {
	// Get standings sorted by position
	standings, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return err
	}
	if len(standings) < 2 {
		return errors.New("not enough players in standings")
	}

	// Clamp topK к реальному числу участников; больше НЕ округляем вниз.
	if topK <= 0 || topK > len(standings) {
		topK = len(standings)
	}
	if topK < 2 {
		return errors.New("need at least 2 teams for playoff")
	}

	// Сид i (1-based) → participants[i-1] по позиции в таблице.
	participants := make([]int64, topK)
	for i := 0; i < topK; i++ {
		participants[i] = standings[i].UserID
	}

	// Расстановка по сетке (стандартный посев + распределённые bye) — общий код.
	slots, matches := buildSeededBracket(leagueID, participants)

	// Слоты и матчи создаются атомарно под advisory-lock лиги: параллельный
	// второй вызов получит ErrBracketExists вместо дублей.
	return s.bracketRepo.GenerateBracket(ctx, leagueID, slots, matches)
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
