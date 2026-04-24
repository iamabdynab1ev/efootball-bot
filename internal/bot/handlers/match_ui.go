package handlers

import (
	"context"
	"efootball-bot/internal/models"
	"fmt"
	"log"
	"math/rand"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleResult(ctx context.Context, msg *tgbotapi.Message, mID int64, hg, ag int16) {
	// Вызываем сервис (в нем теперь проверки на хозяина поля)
	match, err := h.matchSvc.ClaimResult(ctx, mID, msg.From.ID, hg, ag)
	if err != nil {
		h.send(msg.Chat.ID, "❌ Хатолик: "+err.Error())
		return
	}

	aU, _ := h.userRepo.GetByID(ctx, match.AwayUserID)
	if aU == nil {
		return
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnAgree, fmt.Sprintf("confirm_%d", mID)),
			tgbotapi.NewInlineKeyboardButtonData(BtnDisagree, fmt.Sprintf("dispute_%d", mID)),
		),
	)
	text := fmt.Sprintf("⚽ <b>%s</b> натижа киритди: <b>%d-%d</b>. Тасдиқлайсизми?", safe(match.HomeUser.DisplayName), hg, ag)
	notify := tgbotapi.NewMessage(aU.TelegramID, text)
	notify.ParseMode = "HTML"
	notify.ReplyMarkup = kb
	h.bot.Send(notify)
}
func (h *Handler) confirmMatch(ctx context.Context, chatID int64, matchID int64) {
	match, err := h.matchSvc.Confirm(ctx, matchID)
	if err != nil {
		h.send(chatID, "❌ Хатолик: "+err.Error())
		return
	}

	successMsg := fmt.Sprintf("✅ Натижа тасдиқланди!\n\n<b>%s %d:%d %s</b>\n\nЖадвал янгиланди!",
		safe(match.HomeUser.DisplayName), *match.HomeGoals, *match.AwayGoals, safe(match.AwayUser.DisplayName))

	if lastID, ok := h.menuMsgID.Load(chatID); ok {
		h.smartUpdate(chatID, lastID.(int), successMsg, tgbotapi.InlineKeyboardMarkup{})
	} else {
		h.send(chatID, successMsg)
	}

	// ELO янгилаш
	homeUser, _ := h.userRepo.GetByID(ctx, match.HomeUserID)
	awayUser, _ := h.userRepo.GetByID(ctx, match.AwayUserID)
	if homeUser != nil && awayUser != nil {
		homeNew, awayNew := h.eloSvc.Calculate(
			ctx,
			match.HomeUserID, match.AwayUserID,
			homeUser.Rating, awayUser.Rating,
			homeUser.TeamPower, awayUser.TeamPower,
			*match.HomeGoals, *match.AwayGoals,
		)
		_ = h.userRepo.UpdateRating(ctx, homeUser.ID, homeNew)
		_ = h.userRepo.UpdateRating(ctx, awayUser.ID, awayNew)

		// Барча ўйинчилар рангини қайта ҳисоблаш
		go func() {
			_ = h.userRepo.RecalculateAllRanks(context.Background())
		}()

		homeDiff := homeNew - homeUser.Rating
		awayDiff := awayNew - awayUser.Rating

		homeSign := "+"
		if homeDiff < 0 {
			homeSign = ""
		}
		awaySign := "+"
		if awayDiff < 0 {
			awaySign = ""
		}

		homeMsg := tgbotapi.NewMessage(homeUser.TelegramID,
			fmt.Sprintf("📊 <b>Рейтинг:</b> %d → %d (<b>%s%d</b>)\n🏅 %s",
				homeUser.Rating, homeNew, homeSign, homeDiff, homeUser.Rank))
		homeMsg.ParseMode = "HTML"
		h.bot.Send(homeMsg)

		awayMsg := tgbotapi.NewMessage(awayUser.TelegramID,
			fmt.Sprintf("📊 <b>Рейтинг:</b> %d → %d (<b>%s%d</b>)\n🏅 %s",
				awayUser.Rating, awayNew, awaySign, awayDiff, awayUser.Rank))
		awayMsg.ParseMode = "HTML"
		h.bot.Send(awayMsg)
	}

	// Гуруҳга трансляция
	if h.groupID != 0 {
		go func(m *models.Match) {
			league, _ := h.leagueRepo.GetByID(context.Background(), m.LeagueID)
			leagueName := "Лига"
			if league != nil {
				leagueName = league.Name
			}

			hg := *m.HomeGoals
			ag := *m.AwayGoals
			hName := m.HomeUser.DisplayName
			aName := m.AwayUser.DisplayName

			var template string

			e1 := BroadcastWinEmojis[rand.Intn(len(BroadcastWinEmojis))]
			e2 := BroadcastWinEmojis[rand.Intn(len(BroadcastWinEmojis))]
			e3 := BroadcastWinEmojis[rand.Intn(len(BroadcastWinEmojis))]

			if hg == ag {
				de1 := BroadcastDrawEmojis[rand.Intn(len(BroadcastDrawEmojis))]
				de2 := BroadcastDrawEmojis[rand.Intn(len(BroadcastDrawEmojis))]
				phrase := BroadcastDrawPhrases[rand.Intn(len(BroadcastDrawPhrases))]
				template = fmt.Sprintf("%s%s <b>%s %d:%d %s</b> %s", de1, de2, hName, hg, ag, aName, phrase)
			} else {
				var winner, loser string
				var wg, lg int16
				if hg > ag {
					winner, loser, wg, lg = hName, aName, hg, ag
				} else {
					winner, loser, wg, lg = aName, hName, ag, hg
				}
				phrase := BroadcastWinPhrases[rand.Intn(len(BroadcastWinPhrases))]
				template = fmt.Sprintf("%s%s%s <b>%s %d:%d</b> <b>%s</b> %s", e1, e2, e3, winner, wg, lg, loser, phrase)
			}

			broadcastText := fmt.Sprintf("🏆 <b>%s</b>\n\n%s", safe(leagueName), template)

			msg := tgbotapi.NewMessage(h.groupID, broadcastText)
			msg.ParseMode = "HTML"
			h.bot.Send(msg)
		}(match)
	}
}
func (h *Handler) disputeMatch(ctx context.Context, chatID int64, mID int64) {
	match, err := h.matchRepo.GetByID(ctx, mID)
	if err != nil || match == nil {
		h.send(chatID, MsgMatchNotFound)
		return
	}

	// Используем сервис — там есть проверка статуса
	_, err = h.matchSvc.Dispute(ctx, mID, 0, 0)
	if err != nil {
		h.send(chatID, "❌ Хатолик: "+err.Error())
		return
	}

	// Уведомляем хозяина что гость не согласен
	if match.HomeUser != nil {
		notify := tgbotapi.NewMessage(match.HomeUser.TelegramID,
			fmt.Sprintf(MsgDisputeHome, mID))
		notify.ParseMode = "HTML"
		h.bot.Send(notify)
	}

	h.send(chatID, MsgDisputeSent)
}
func (h *Handler) editScoreMessage(ctx context.Context, chatID int64, msgID int, telegramID int64, matchID int64, hg, ag int16) {
	// 2. Получаем данные матча
	m, err := h.matchRepo.GetByID(ctx, matchID)
	if err != nil || m == nil || m.HomeUser == nil || m.AwayUser == nil {
		log.Printf("❌ Хато: Ўйин маълумотларини юклаб бўлмади (ID: %d)", matchID)
		return
	}

	// 3. Подготовка имен
	hName := safe(m.HomeUser.DisplayName)
	aName := safe(m.AwayUser.DisplayName)

	// 4. Формируем текст сообщения (HTML режим)
	text := fmt.Sprintf("⚽ <b>Ҳисобни белгиланг:</b>\n\n🏠 %s [ <b>%d : %d</b> ] %s ✈️",
		hName, hg, ag, aName)

	// 5. Создаем клавиатуру
	kb := tgbotapi.NewInlineKeyboardMarkup(
		// Ряд для домашней команды (Хозяин)
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnMinus, fmt.Sprintf("s_mod_%d_%d_%d", matchID, hg-1, ag)),
			tgbotapi.NewInlineKeyboardButtonData(hName, "ignore"), // Имя игрока как ярлык
			tgbotapi.NewInlineKeyboardButtonData(BtnPlus, fmt.Sprintf("s_mod_%d_%d_%d", matchID, hg+1, ag)),
		),
		// Ряд для гостевой команды (Меҳмон)
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnMinus, fmt.Sprintf("s_mod_%d_%d_%d", matchID, hg, ag-1)),
			tgbotapi.NewInlineKeyboardButtonData(aName, "ignore"), // Имя игрока как ярлык
			tgbotapi.NewInlineKeyboardButtonData(BtnPlus, fmt.Sprintf("s_mod_%d_%d_%d", matchID, hg, ag+1)),
		),
		// Ряд подтверждения
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Юбориш (%d:%d)", hg, ag), fmt.Sprintf("s_sub_%d_%d_%d", matchID, hg, ag)),
		),
		// Ряд возврата
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "schedule"),
		),
	)

	h.smartUpdate(chatID, msgID, text, kb)
}
func (h *Handler) HandleConfirm(ctx context.Context, msg *tgbotapi.Message, mID int64) {
	h.confirmMatch(ctx, msg.Chat.ID, mID)
}
func (h *Handler) HandleDispute(ctx context.Context, msg *tgbotapi.Message, mID int64) {
	h.disputeMatch(ctx, msg.Chat.ID, mID)
}
func (h *Handler) sendMatchesForScore(ctx context.Context, chatID int64, msgID int, telegramID int64) {
	u, _ := h.userRepo.GetByTelegramID(ctx, telegramID)
	leagues, _ := h.leagueRepo.GetActiveLeagues(ctx)
	var active []*models.Match
	for _, l := range leagues {
		ms, _ := h.matchRepo.GetUserSchedule(ctx, u.ID, l.ID)
		for _, m := range ms {
			if m.HomeUserID == u.ID && m.Status == models.MatchScheduled {
				active = append(active, m)
			}
		}
	}
	if len(active) == 0 {
		h.smartUpdate(chatID, msgID, MsgNoMatchesToScore, tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"))))
		return
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, m := range active {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d-тур vs %s", m.Round, m.AwayUser.DisplayName), fmt.Sprintf("enter_match_%d", m.ID))))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")))
	h.smartUpdate(chatID, msgID, MsgSelectMatch, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
