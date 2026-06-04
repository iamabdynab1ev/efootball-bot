package handlers

import (
	"context"
	"efootball-bot/internal/cardgen"
	"efootball-bot/internal/i18n"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) sendPlayerCard(ctx context.Context, chatID, telegramID int64) {
	user, err := h.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.send(chatID, i18n.T("uz", "not.registered"))
		return
	}

	lang := user.Language

	// Collect achievements (up to 5 icons)
	icons := []string{}
	if h.achievRepo != nil {
		achievements, err := h.achievRepo.GetUserAchievements(ctx, user.ID)
		if err == nil {
			for _, ua := range achievements {
				if ua.Achievement != nil {
					icons = append(icons, ua.Achievement.Icon)
				}
				if len(icons) >= 5 {
					break
				}
			}
		}
	}

	// Build global stats from match repo
	wins, draws, losses := 0, 0, 0
	totalGoals := 0
	if stats, err := h.userRepo.GetGlobalStats(ctx, user.ID); err == nil && stats != nil {
		wins = stats.TotalWins
		draws = stats.TotalDraws
		losses = stats.TotalLosses
		totalGoals = stats.TotalGoalsFor
	}

	played := wins + draws + losses
	winRate := 0.0
	if played > 0 {
		winRate = float64(wins) / float64(played) * 100
	}

	club := ""
	if user.FavoriteClub != nil {
		club = *user.FavoriteClub
	}

	data := cardgen.CardData{
		DisplayName:  user.DisplayName,
		Rank:         user.Rank,
		Rating:       int(user.Rating),
		TeamPower:    user.TeamPower,
		FavoriteClub: club,
		Wins:         wins,
		Draws:        draws,
		Losses:       losses,
		TotalGoals:   totalGoals,
		WinRate:      winRate,
		Achievements: icons,
	}

	pngBytes, err := cardgen.GenerateCard(data)
	if err != nil {
		h.send(chatID, "❌ Карта яратишда хатолик юз берди.")
		return
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
		Name:  "card.png",
		Bytes: pngBytes,
	})
	photo.Caption = fmt.Sprintf(i18n.T(lang, "card.caption"), user.DisplayName, user.Rating, "")
	photo.ParseMode = "HTML"
	h.sendChattable(photo)
}
