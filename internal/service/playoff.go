package service

import (
	"context"
	"efootball-bot/internal/engine"
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
	seedUser := func(seed int) int64 { return participants[seed-1] }

	// Полный размер сетки — ближайшая степень двойки ≥ topK.
	size := models.NextPowerOf2(topK)
	firstStage := models.FirstStageForSize(size)
	secondStage := models.NextStage(firstStage)

	firstRound, byes := engine.FirstRound(topK)

	// ── Слоты ──────────────────────────────────────────────────────────────
	var allSlots []*models.BracketSlot
	// Индекс пустых слотов следующих стадий, чтобы проставить bye-сидов.
	slotIndex := map[string]map[int]*models.BracketSlot{}

	// Первая стадия: только реальные матчи (bye-сиды сюда не попадают).
	for _, m := range firstRound {
		home := seedUser(m.HomeSeed)
		away := seedUser(m.AwaySeed)
		allSlots = append(allSlots, &models.BracketSlot{
			LeagueID:   leagueID,
			Stage:      firstStage,
			Slot:       m.Slot,
			HomeUserID: &home,
			AwayUserID: &away,
		})
	}

	// Последующие стадии: пустые слоты (заполняются по мере продвижения).
	matchesInStage := size / 2
	for stage := secondStage; stage != ""; stage = models.NextStage(stage) {
		count := matchesInStage / 2
		if count < 1 {
			count = 1
		}
		slotIndex[stage] = map[int]*models.BracketSlot{}
		for i := 1; i <= count; i++ {
			sl := &models.BracketSlot{LeagueID: leagueID, Stage: stage, Slot: i}
			allSlots = append(allSlots, sl)
			slotIndex[stage][i] = sl
		}
		matchesInStage = count
	}

	// Bye: топ-сид заранее ставится в слот второй стадии (без матча).
	for _, b := range byes {
		target := slotIndex[secondStage][b.NextSlot]
		if target == nil {
			continue // защита от рассинхронизации; в норме слот существует
		}
		uid := seedUser(b.Seed)
		if b.IsHome {
			target.HomeUserID = &uid
		} else {
			target.AwayUserID = &uid
		}
	}

	// ── Матчи первой стадии ─────────────────────────────────────────────────
	var matches []*models.Match
	roundNum := int16(100) // нумерация со 100 — отличать от лиговых раундов
	for _, m := range firstRound {
		home := seedUser(m.HomeSeed)
		away := seedUser(m.AwaySeed)
		slotCopy := m.Slot
		matches = append(matches, &models.Match{
			LeagueID:    leagueID,
			HomeUserID:  home,
			AwayUserID:  away,
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
