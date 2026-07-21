package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"efootball-bot/internal/i18n"
	"efootball-bot/internal/logger"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
)

// SeasonService — закрытие сезона с церемонией: подводит итоги по всем лигам,
// раздаёт сезонные номинации, открывает следующий сезон, зовёт всех на
// церемонию и публикует итоговый пост в группу.
//
// Номинации сезона (league_id NULL в season_awards):
//   - season_player      — Игрок сезона: сумма очков по лигам + бонусы за медали
//     (чемпион +10, серебро +6, бронза +4);
//   - season_top_scorer  — Бомбардир сезона: сумма голов;
//   - season_best_defense — Стена сезона: меньше всех пропущено (при равенстве — больше очков);
//   - season_elo_growth  — Прорыв сезона: наибольший рост ELO за сезон.
type SeasonService struct {
	leagueRepo repository.LeagueRepository
	awardRepo  repository.AwardRepository
	userRepo   repository.UserRepository
	predRepo   repository.PredictionRepository // может быть nil — Оракул не разыгрывается
	notif      *NotificationService
	groups     GroupPublisher
}

func NewSeasonService(lr repository.LeagueRepository, ar repository.AwardRepository, ur repository.UserRepository) *SeasonService {
	return &SeasonService{leagueRepo: lr, awardRepo: ar, userRepo: ur}
}

// SetPredictions подключает прогнозы — номинация «🔮 Оракул сезона».
func (s *SeasonService) SetPredictions(p repository.PredictionRepository) { s.predRepo = p }

func (s *SeasonService) SetNotifications(n *NotificationService) { s.notif = n }
func (s *SeasonService) SetGroups(g GroupPublisher)              { s.groups = g }

// medalBonus — вес медалей лиг в зачёте «Игрока сезона».
var medalBonus = map[string]int{"champion": 10, "runner_up": 6, "third_place": 4}

// Nomination — итог одной номинации (для ответа API и постов).
type Nomination struct {
	Type  string `json:"type"`
	User  int64  `json:"user_id"`
	Name  string `json:"name"`
	Club  string `json:"club"`
	Value int    `json:"value"`
}

// Close закрывает сезон: валидация → номинации → награды → новый сезон →
// уведомления и групповой пост. Возвращает номинации и следующий сезон.
func (s *SeasonService) Close(ctx context.Context, seasonID int64, nextName string) ([]Nomination, *models.Season, error) {
	season, err := s.leagueRepo.GetSeasonByID(ctx, seasonID)
	if err != nil || season == nil {
		return nil, nil, fmt.Errorf("сезон не найден")
	}
	if season.Status != "active" {
		return nil, nil, fmt.Errorf("сезон уже закрыт")
	}

	leagues, err := s.leagueRepo.ListLeaguesBySeason(ctx, seasonID)
	if err != nil {
		return nil, nil, fmt.Errorf("лиги сезона: %w", err)
	}
	var unfinished []string
	hasPlayed := false
	for _, l := range leagues {
		switch l.Status {
		case models.LeagueFinished, models.LeagueArchived:
			hasPlayed = true
		case models.LeagueDraft:
			// пустые черновики не мешают закрытию
		default:
			unfinished = append(unfinished, l.Name)
		}
	}
	if len(unfinished) > 0 {
		return nil, nil, fmt.Errorf("сначала завершите лиги: %s", strings.Join(unfinished, ", "))
	}
	if !hasPlayed {
		return nil, nil, fmt.Errorf("в сезоне нет ни одной завершённой лиги")
	}

	noms, err := s.computeNominations(ctx, season)
	if err != nil {
		return nil, nil, err
	}

	// Награды сезона — идемпотентно (повторное закрытие обновит, не задвоит).
	for _, n := range noms {
		if _, aErr := s.awardRepo.CreateSeasonAward(ctx, seasonID, n.Type, n.User, n.Value); aErr != nil {
			logger.FromContext(ctx).Error("season award", "type", n.Type, "err", aErr)
		}
	}

	if strings.TrimSpace(nextName) == "" {
		nextName = fmt.Sprintf("Сезон %d", seasonID+1)
	}
	next, err := s.leagueRepo.CloseSeason(ctx, seasonID, nextName)
	if err != nil {
		return nil, nil, fmt.Errorf("закрыть сезон: %w", err)
	}

	s.announce(ctx, season, noms)
	return noms, next, nil
}

// computeNominations считает четыре сезонные номинации.
func (s *SeasonService) computeNominations(ctx context.Context, season *models.Season) ([]Nomination, error) {
	aggs, err := s.leagueRepo.SeasonAggregates(ctx, season.ID)
	if err != nil {
		return nil, fmt.Errorf("агрегаты сезона: %w", err)
	}
	if len(aggs) == 0 {
		return nil, fmt.Errorf("в сезоне нет участников")
	}

	// Медальные бонусы из уже выданных лиговых наград сезона.
	bonuses := map[int64]int{}
	if awards, aErr := s.awardRepo.GetBySeason(ctx, season.ID); aErr == nil {
		for _, a := range awards {
			if a.LeagueID != nil {
				bonuses[a.UserID] += medalBonus[a.AwardType]
			}
		}
	}

	deltas, _ := s.leagueRepo.EloDeltasSince(ctx, season.CreatedAt)

	// Игрок сезона: очки + бонусы; тай-брейк — разница, затем голы.
	player := aggs[0]
	score := func(a *repository.SeasonAggregate) int { return a.Points + bonuses[a.UserID] }
	for _, a := range aggs[1:] {
		as, ps := score(a), score(player)
		agd, pgd := a.GoalsFor-a.GoalsAgainst, player.GoalsFor-player.GoalsAgainst
		if as > ps || (as == ps && (agd > pgd || (agd == pgd && a.GoalsFor > player.GoalsFor))) {
			player = a
		}
	}

	// Бомбардир: больше всех голов; тай-брейк — меньше матчей.
	scorer := aggs[0]
	played := func(a *repository.SeasonAggregate) int { return a.Wins + a.Draws + a.Losses }
	for _, a := range aggs[1:] {
		if a.GoalsFor > scorer.GoalsFor || (a.GoalsFor == scorer.GoalsFor && played(a) < played(scorer)) {
			scorer = a
		}
	}

	// Стена сезона: меньше всех пропустил среди реально игравших; тай-брейк — очки.
	var wall *repository.SeasonAggregate
	for _, a := range aggs {
		if played(a) == 0 {
			continue
		}
		if wall == nil || a.GoalsAgainst < wall.GoalsAgainst ||
			(a.GoalsAgainst == wall.GoalsAgainst && a.Points > wall.Points) {
			wall = a
		}
	}

	// Прорыв сезона: наибольший рост ELO среди участников сезона.
	var breakout *repository.SeasonAggregate
	bestDelta := 0
	for _, a := range aggs {
		if d, ok := deltas[a.UserID]; ok && (breakout == nil || d > bestDelta) && d > 0 {
			breakout, bestDelta = a, d
		}
	}

	noms := []Nomination{
		{Type: "season_player", User: player.UserID, Name: player.DisplayName, Club: player.FavoriteClub, Value: score(player)},
		{Type: "season_top_scorer", User: scorer.UserID, Name: scorer.DisplayName, Club: scorer.FavoriteClub, Value: scorer.GoalsFor},
	}
	if wall != nil {
		noms = append(noms, Nomination{Type: "season_best_defense", User: wall.UserID, Name: wall.DisplayName, Club: wall.FavoriteClub, Value: wall.GoalsAgainst})
	}
	if breakout != nil {
		noms = append(noms, Nomination{Type: "season_elo_growth", User: breakout.UserID, Name: breakout.DisplayName, Club: breakout.FavoriteClub, Value: bestDelta})
	}

	// 🔮 Оракул сезона: лучший прогнозист по сумме очков за лиги сезона.
	if s.predRepo != nil {
		if pts, pErr := s.predRepo.SeasonPoints(ctx, season.ID); pErr == nil && len(pts) > 0 {
			var bestUID int64
			best := 0
			for uid, p := range pts {
				if p > best || (p == best && bestUID != 0 && uid < bestUID) {
					bestUID, best = uid, p
				}
			}
			if bestUID != 0 && best > 0 {
				name, club := "—", ""
				for _, a := range aggs {
					if a.UserID == bestUID {
						name, club = a.DisplayName, a.FavoriteClub
						break
					}
				}
				if name == "—" {
					if u, uErr := s.userRepo.GetByID(ctx, bestUID); uErr == nil && u != nil {
						name = u.DisplayName
					}
				}
				noms = append(noms, Nomination{Type: "season_oracle", User: bestUID, Name: name, Club: club, Value: best})
			}
		}
	}
	return noms, nil
}

// announce — персональные уведомления участникам + итоговый пост в группу.
func (s *SeasonService) announce(ctx context.Context, season *models.Season, noms []Nomination) {
	aggs, err := s.leagueRepo.SeasonAggregates(ctx, season.ID)
	if err != nil {
		return
	}
	ids := make([]int64, 0, len(aggs))
	for _, a := range aggs {
		ids = append(ids, a.UserID)
	}
	link := fmt.Sprintf("/season?id=%d", season.ID)
	if s.notif != nil {
		s.notif.NotifyT(ctx, ids, models.NotifAward, link, func(lang string) (string, string) {
			return fmt.Sprintf(i18n.T(lang, "season.closed.title"), season.Name),
				i18n.T(lang, "season.closed.body")
		})
	}

	if s.groups != nil {
		matches, goals, _ := s.leagueRepo.SeasonTotals(ctx, season.ID)
		var b strings.Builder
		fmt.Fprintf(&b, "🏆✨ *СЕЗОН «%s» ЗАВЕРШЁН!* ✨🏆\n\n", season.Name)
		fmt.Fprintf(&b, "📊 За сезон: *%d матчей*, *%d голов*\n\n*Герои сезона:*\n", matches, goals)
		labels := map[string]string{
			"season_player":       "👑 Игрок сезона",
			"season_top_scorer":   "⚽ Бомбардир сезона",
			"season_best_defense": "🧱 Стена сезона",
			"season_elo_growth":   "🚀 Прорыв сезона",
			"season_oracle":       "🔮 Оракул сезона",
		}
		order := []string{"season_player", "season_top_scorer", "season_best_defense", "season_elo_growth", "season_oracle"}
		byType := map[string]Nomination{}
		for _, n := range noms {
			byType[n.Type] = n
		}
		for _, tpe := range order {
			if n, ok := byType[tpe]; ok {
				fmt.Fprintf(&b, "%s — *%s* (%d)\n", labels[tpe], n.Name, n.Value)
			}
		}
		b.WriteString("\nЦеремония награждения — в приложении. Новый сезон уже открыт! 🎉")
		s.groups.Publish(b.String())
	}
}

// Summary — данные церемонии: сезон, итоги, номинации, чемпионы лиг.
func (s *SeasonService) Summary(ctx context.Context, seasonID int64) (map[string]any, error) {
	season, err := s.leagueRepo.GetSeasonByID(ctx, seasonID)
	if err != nil || season == nil {
		return nil, fmt.Errorf("сезон не найден")
	}
	matches, goals, _ := s.leagueRepo.SeasonTotals(ctx, seasonID)
	aggs, _ := s.leagueRepo.SeasonAggregates(ctx, seasonID)

	awards, err := s.awardRepo.GetBySeason(ctx, seasonID)
	if err != nil {
		return nil, err
	}
	clubOf := map[int64]string{}
	for _, a := range aggs {
		clubOf[a.UserID] = a.FavoriteClub
	}

	var nominations []map[string]any
	var champions []map[string]any
	for _, a := range awards {
		val := 0
		if a.Value != nil {
			val = *a.Value
		}
		if a.LeagueID == nil {
			nominations = append(nominations, map[string]any{
				"type": a.AwardType, "user_id": a.UserID, "name": a.UserDisplayName,
				"club": clubOf[a.UserID], "value": val,
			})
		} else if a.AwardType == "champion" {
			champions = append(champions, map[string]any{
				"league_id": *a.LeagueID, "league_name": a.LeagueName,
				"user_id": a.UserID, "name": a.UserDisplayName, "club": clubOf[a.UserID],
			})
		}
	}
	// Номинации в сценарном порядке церемонии.
	orderIdx := map[string]int{"season_oracle": 0, "season_elo_growth": 1, "season_best_defense": 2, "season_top_scorer": 3, "season_player": 4}
	sort.SliceStable(nominations, func(i, j int) bool {
		return orderIdx[nominations[i]["type"].(string)] < orderIdx[nominations[j]["type"].(string)]
	})
	sort.SliceStable(champions, func(i, j int) bool {
		return champions[i]["league_id"].(int64) < champions[j]["league_id"].(int64)
	})

	var closedAt string
	if season.ClosedAt != nil {
		closedAt = season.ClosedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	return map[string]any{
		"season": map[string]any{
			"id": season.ID, "name": season.Name, "status": season.Status, "closed_at": closedAt,
		},
		"totals":      map[string]any{"matches": matches, "goals": goals, "players": len(aggs)},
		"nominations": nominations,
		"champions":   champions,
	}, nil
}
