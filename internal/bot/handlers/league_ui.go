package handlers

import (
	"context"
	"efootball-bot/internal/models"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) sendTableUI(ctx context.Context, chatID int64, msgID int, lID int64) {
	league, err := h.leagueRepo.GetByID(ctx, lID)
	if err != nil || league == nil {
		h.smartUpdate(chatID, msgID, MsgLeagueNotFound, tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "table")),
		))
		return
	}

	members, err := h.leagueRepo.GetMembers(ctx, lID)
	if err != nil {
		h.smartUpdate(chatID, msgID, "❌ Хатолик юз берди.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "table")),
		))
		return
	}

	text := fmt.Sprintf("📊 <b>%s — Турнир жадвали</b>\n\n", safe(league.Name))

	if len(members) == 0 {
		h.smartUpdate(chatID, msgID, text+"📊 Жадвал бўш.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "table")),
		))
		return
	}

	// ЖАДВАЛНИНГ ТЕППА ҚИСМИ (Аббревиатуралар билан)
	// Ў-Ўйин, Ғ-Ғалаба, Д-Дуранг, М-Мағлубият, +-Урилган, --Ўтказилган, О-Очко
	text += "<pre>"
	text += fmt.Sprintf("%-2s %-10s %2s %1s %1s %1s %2s %2s %2s\n", "№", "Жамоа", "Ў", "Ғ", "Д", "М", "+", "-", "О")
	text += "──────────────────────────────\n"

	for i, m := range members {
		name := "Player"
		if m.User != nil {
			name = m.User.DisplayName
		}

		// Телефон экрани учун исмни 10 та белгигача қирқамиз (текис туриши учун)
		nameRunes := []rune(name)
		if len(nameRunes) > 10 {
			name = string(nameRunes[:9]) + "…"
		}

		// Ўйинчи номидаги HTML белгиларни тозалаймиз
		name = strings.ReplaceAll(name, "<", "")
		name = strings.ReplaceAll(name, ">", "")

		played := m.Wins + m.Draws + m.Losses

		// МАЪЛУМОТЛАРНИ УСТУНЛАРГА ЖОЙЛАШТИРИШ
		// %-2d (№) %-10s (Жамоа) %2d (Ўйин) %1d (Ғ) %1d (Д) %1d (М) %2d (+) %2d (-) %2d (Очко)
		text += fmt.Sprintf("%-2d %-10s %2d %1d %1d %1d %2d %2d %2d\n",
			i+1, name, played, m.Wins, m.Draws, m.Losses, m.GoalsFor, m.GoalsAgainst, m.Points)
	}
	text += "</pre>"

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnRefresh, fmt.Sprintf("table_%d", lID)),
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "table"),
		),
	)

	h.smartUpdate(chatID, msgID, text, kb)
}
func (h *Handler) sendTableSelectorUI(ctx context.Context, chatID int64, msgID int) {
	leagues, _ := h.leagueRepo.GetActiveLeagues(ctx)
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, l := range leagues {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(l.Name, fmt.Sprintf("table_%d", l.ID))))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")))
	h.smartUpdate(chatID, msgID, MsgChooseLeague2, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) sendScheduleSelectorUI(ctx context.Context, chatID int64, msgID int, tgID int64) {
	u, _ := h.userRepo.GetByTelegramID(ctx, tgID)
	if u == nil {
		return
	}

	ls, _ := h.leagueRepo.GetActiveLeagues(ctx)
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, l := range ls {
		// Проверяем участие (любой статус кроме rejected)
		is, _ := h.leagueRepo.IsMember(ctx, l.ID, u.ID)
		if is {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📅 "+l.Name, "sched_view_"+fmt.Sprint(l.ID)),
			))
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")))

	text := "📅 <b>Чизмани кўриш учун лигани танланг:</b>"
	if len(rows) <= 1 {
		text = "😔 Сиз ҳали ҳеч қайси лигага аъзо эмассиз ёки ариза бермагансиз."
	}

	h.smartUpdate(chatID, msgID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) sendScheduleByLeagueUI(ctx context.Context, chatID int64, msgID int, tgID int64, lID int64) {
	u, _ := h.userRepo.GetByTelegramID(ctx, tgID)
	if u == nil {
		return
	}

	l, _ := h.leagueRepo.GetByID(ctx, lID)
	ms, _ := h.matchRepo.GetUserSchedule(ctx, u.ID, lID)

	// Сарлавҳа
	text := fmt.Sprintf("📅 <b>%s — Ўйинлар чизмаси:</b>\n\n", safe(l.Name))

	if l.Status == models.LeagueRegistration {
		h.smartUpdate(chatID, msgID, "🏁 Бу лигада ҳали қуръа ташланмаган.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "schedule")),
		))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	btnF := false
	uName := safe(u.DisplayName)

	if len(ms) == 0 {
		text += "✅ Сизда бу лигада ўйналмаган ўйинлар қолмади."
	} else {
		for _, m := range ms {
			hName, aName := safe(m.HomeUser.DisplayName), safe(m.AwayUser.DisplayName)

			if m.HomeUserID == u.ID {
				// УЙДАГИ ЎЙИН
				if !btnF {
					rows = append(rows, tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📝 %d-тур натижасини ёзиш", m.Round), fmt.Sprintf("enter_match_%d", m.ID)),
					))
					btnF = true
					// Тартиб: ➡️ (Уйда) 1-тур: [Сиз] vs [Рақиб]
					text += fmt.Sprintf("➡️ (Уйда) <b>%d-тур:</b> <b>%s</b> 🆚 %s\n", m.Round, uName, aName)
				} else {
					text += fmt.Sprintf("(Уйда) <b>%d-тур:</b> <b>%s</b> 🆚 %s\n", m.Round, uName, aName)
				}
			} else {
				// САФАРДАГИ ЎЙИН
				// Тартиб: (Сафарда) 1-тур: [Рақиб] vs [Сиз]
				text += fmt.Sprintf("(Сафарда) <b>%d-тур:</b> %s 🆚 <b>%s</b>\n", m.Round, hName, uName)
			}
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "schedule")))
	h.smartUpdate(chatID, msgID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) sendSchedule(ctx context.Context, chatID int64, msgID int, telegramID int64) {
	user, _ := h.userRepo.GetByTelegramID(ctx, telegramID)
	if user == nil {
		return
	}

	// Один запрос вместо цикла по всем лигам
	memberships, _ := h.leagueRepo.GetUserLeagues(ctx, user.ID)

	if len(memberships) == 0 {
		h.smartUpdate(chatID, msgID, "😔 Сиз ҳали ҳеч қайси лигада йўқсиз.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")),
		))
		return
	}

	// Если одна лига — сразу показываем расписание
	if len(memberships) == 1 {
		h.sendScheduleByLeagueUI(ctx, chatID, msgID, telegramID, memberships[0].LeagueID)
		return
	}

	// Если несколько лиг — показываем выбор
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, m := range memberships {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 "+m.League.Name, fmt.Sprintf("sched_view_%d", m.LeagueID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
	))

	h.smartUpdate(chatID, msgID, "📅 <b>Лигани танланг:</b>", tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) sendLeagueList(ctx context.Context, chatID int64, msgID int) {
	leagues, err := h.leagueRepo.GetActiveLeagues(ctx)
	if err != nil || len(leagues) == 0 {
		h.smartUpdate(chatID, msgID, MsgNoLeagues, tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")),
		))
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, l := range leagues {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏴 "+l.Name, fmt.Sprintf("join_%d", l.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
	))

	h.smartUpdate(chatID, msgID, MsgChooseLeague, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) joinLeagueByID(ctx context.Context, chatID int64, telegramID int64, lID int64, msgID int) {
	u, err := h.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil || u == nil {
		h.send(chatID, MsgNotRegistered)
		return
	}

	l, err := h.leagueRepo.GetByID(ctx, lID)
	if err != nil || l == nil {
		h.smartUpdate(chatID, msgID, MsgLeagueNotFound, tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")),
		))
		return
	}

	status, err := h.leagueRepo.GetMemberStats(ctx, l.ID, u.ID)
	if err == nil && status != nil {
		if status.Status == "approved" {
			h.smartUpdate(chatID, msgID, fmt.Sprintf("✅ Сиз <b>%s</b> лигасида аллақачон қатнашяпсиз!", l.Name), tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")),
			))
			return
		}
		if status.Status == "pending" {
			h.smartUpdate(chatID, msgID, fmt.Sprintf("⏳ <b>%s</b> лигасига аризангиз кўриб чиқилмоқда.", l.Name), tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")),
			))
			return
		}
	}

	err = h.leagueRepo.AddMember(ctx, l.ID, u.ID)
	if err != nil {
		h.send(chatID, "❌ Ариза юборишда хатолик юз берди.")
		return
	}
	successText := fmt.Sprintf(MsgApplicationSent, safe(l.Name))
	h.smartUpdate(chatID, msgID, successText, tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")),
	))
}
func (h *Handler) HandleJoin(ctx context.Context, msg *tgbotapi.Message, lName string) {
	u, err := h.userRepo.GetByTelegramID(ctx, msg.From.ID)
	if err != nil || u == nil {
		h.send(msg.Chat.ID, MsgNotRegistered)
		return
	}
	l, err := h.leagueRepo.GetByName(ctx, lName)
	if err != nil || l == nil {
		h.send(msg.Chat.ID, MsgLeagueNotFound)
		return
	}
	h.joinLeagueByID(ctx, msg.Chat.ID, msg.From.ID, l.ID, 0)
}
func (h *Handler) sendMatchHistory(ctx context.Context, chatID int64, msgID int, telegramID int64) {
	user, _ := h.userRepo.GetByTelegramID(ctx, telegramID)
	if user == nil {
		return
	}

	// Ўйнаб бўлинган ўйинларни оламиз
	history, err := h.matchRepo.GetUserMatchHistory(ctx, user.ID)
	if err != nil || len(history) == 0 {
		h.smartUpdate(chatID, msgID, "📜 Сизда ҳали тугалланган ўйинлар мавжуд эмас.", tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"))))
		return
	}

	text := "📜 <b>Сизнинг ўйинлар тарихингиз:</b>\n\n"
	for _, m := range history {
		hN := m.HomeUser.DisplayName
		aN := m.AwayUser.DisplayName
		hg := m.GetHomeGoals()
		ag := m.GetAwayGoals()

		// Ўйинчи ўз исмини дарров топиши учун қавс ичига оламиз ёки жирн қиламиз
		if m.HomeUserID == user.ID {
			hN = "<b>" + hN + "</b>"
		} else {
			aN = "<b>" + aN + "</b>"
		}

		text += fmt.Sprintf("▫️ Тур %d: %s  %d — %d  %s\n", m.Round, hN, hg, ag, aN)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")))
	h.smartUpdate(chatID, msgID, text, kb)
}
func (h *Handler) sendTopScorers(ctx context.Context, chatID int64, msgID int) {
	leagues, _ := h.leagueRepo.GetActiveLeagues(ctx)
	if len(leagues) == 0 {
		h.smartUpdate(chatID, msgID, "😔 Фаол лигалар йўқ.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")),
		))
		return
	}

	text := "👟 <b>Энг яхши тўпурарлар</b>\n"

	for _, l := range leagues {
		scorers, err := h.userRepo.GetTopScorers(ctx, l.ID)
		if err != nil || len(scorers) == 0 {
			continue
		}

		leagueText := ""
		count := 0

		for _, s := range scorers {
			if count >= 10 {
				break
			}
			if s.GoalsFor == 0 {
				continue
			}

			var medal string
			switch count {
			case 0:
				medal = "🥇"
			case 1:
				medal = "🥈"
			case 2:
				medal = "🥉"
			default:
				medal = fmt.Sprintf("%d.", count+1)
			}

			name := s.User.DisplayName
			nameRunes := []rune(name)
			if len(nameRunes) > 15 {
				name = string(nameRunes[:14]) + "…"
			}

			leagueText += fmt.Sprintf("%s <b>%s</b> — %d гол\n", medal, safe(name), s.GoalsFor)
			count++
		}

		if leagueText == "" {
			continue
		}

		text += fmt.Sprintf("\n🏆 <b>%s</b>\n", safe(l.Name))
		text += leagueText
	}

	if text == "👟 <b>Энг яхши тўпурарлар</b>\n" {
		text += "\n😔 Ҳали ҳеч ким гол урмаган!"
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Янгилаш", "top_scorers"),
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
		),
	)
	h.smartUpdate(chatID, msgID, text, kb)
}
func (h *Handler) sendHistoryLeagueSelect(ctx context.Context, chatID int64, msgID int, telegramID int64) {
	user, _ := h.userRepo.GetByTelegramID(ctx, telegramID)
	if user == nil {
		h.send(chatID, MsgNotRegistered)
		return
	}

	memberships, _ := h.leagueRepo.GetUserLeagues(ctx, user.ID)
	if len(memberships) == 0 {
		h.smartUpdate(chatID, msgID, "😔 Сиз ҳали ҳеч қайси лигада ўйнамагансиз.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")),
		))
		return
	}

	// Битта лига бўлса дарров кўрсатамиз
	if len(memberships) == 1 {
		h.sendMatchHistoryByLeague(ctx, chatID, msgID, telegramID, memberships[0].LeagueID)
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, m := range memberships {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"📋 "+m.League.Name,
				fmt.Sprintf("history_league_%d", m.LeagueID),
			),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
	))

	h.smartUpdate(chatID, msgID, "📋 <b>Қайси лига тарихини кўрмоқчисиз?</b>", tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) sendMatchHistoryByLeague(ctx context.Context, chatID int64, msgID int, telegramID int64, leagueID int64) {
	user, _ := h.userRepo.GetByTelegramID(ctx, telegramID)
	if user == nil {
		return
	}

	league, _ := h.leagueRepo.GetByID(ctx, leagueID)
	leagueName := "Лига"
	if league != nil {
		leagueName = league.Name
	}

	matches, _ := h.matchRepo.GetUserSchedule(ctx, user.ID, leagueID)
	if len(matches) == 0 {
		h.smartUpdate(chatID, msgID, "😔 Ҳали ўйинлар йўқ.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "match_history")),
		))
		return
	}

	text := fmt.Sprintf("📋 <b>%s — Ўйинлар тарихи:</b>\n\n", safe(leagueName))

	wins, draws, losses := 0, 0, 0
	goalsFor, goalsAgainst := 0, 0

	for _, m := range matches {
		if m.Status != models.MatchConfirmed {
			continue
		}

		var myGoals, oppGoals int16
		var oppName string

		if m.HomeUserID == user.ID {
			myGoals = *m.HomeGoals
			oppGoals = *m.AwayGoals
			if m.AwayUser != nil {
				oppName = m.AwayUser.DisplayName
			}
		} else {
			myGoals = *m.AwayGoals
			oppGoals = *m.HomeGoals
			if m.HomeUser != nil {
				oppName = m.HomeUser.DisplayName
			}
		}

		goalsFor += int(myGoals)
		goalsAgainst += int(oppGoals)

		var result, emoji string
		if myGoals > oppGoals {
			result = "Ғ"
			emoji = "✅"
			wins++
		} else if myGoals < oppGoals {
			result = "М"
			emoji = "❌"
			losses++
		} else {
			result = "Д"
			emoji = "🤝"
			draws++
		}

		text += fmt.Sprintf("%s <b>%d тур:</b> %s — <b>%d:%d</b> [%s]\n",
			emoji, m.Round, safe(oppName), myGoals, oppGoals, result)
	}

	played := wins + draws + losses
	if played == 0 {
		text += "😔 Ҳали тасдиқланган ўйинлар йўқ."
	} else {
		text += fmt.Sprintf("\n📊 <b>Жами:</b> %d ўйин | ✅%d 🤝%d ❌%d | ⚽%d:🥅%d",
			played, wins, draws, losses, goalsFor, goalsAgainst)
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnBack, "match_history"),
	))

	h.smartUpdate(chatID, msgID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
