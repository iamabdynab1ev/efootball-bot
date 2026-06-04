package handlers

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/service"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) adminDraw(ctx context.Context, chatID int64, lID int64, msgID int) {
	mems, err := h.leagueRepo.GetMembers(ctx, lID)
	if err != nil || len(mems) < 2 {
		errMsg := "❌ <b>Хато:</b> Қуръа ташлаш учун лигада камида 2 та тасдиқланган иштирокчи бўлиши шарт!"
		h.smartUpdate(chatID, msgID, errMsg, tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(BtnBack, "admin_draw"),
			),
		))
		return
	}

	league, err := h.leagueRepo.GetByID(ctx, lID)
	if err != nil || league == nil {
		h.send(chatID, MsgLeagueNotFound)
		return
	}

	cfg := service.Calculate(len(mems), league.RoundsType)

	switch league.RoundsType {
	case "groups", "groups_playoff":
		err = h.groupStageSvc.GenerateGroupStage(ctx, lID, cfg.NumGroups, cfg.GroupAdvance)
	default: // "single", "double"
		double := league.RoundsType == "double"
		err = h.schedSvc.GenerateSchedule(ctx, lID, double)
	}
	if err != nil {
		h.send(chatID, "❌ Ўйинларни яратишда хатолик юз берди.")
		return
	}

	_ = h.leagueRepo.SetLeagueStatus(ctx, lID, string(models.LeagueActive))

	h.notifyPlayersAboutSchedule(ctx, lID, mems)

	h.send(chatID, fmt.Sprintf(
		"✅ <b>%s</b> лигаси бўйича қуръа якунланди!",
		safe(league.Name),
	))
}
func (h *Handler) adminApprove(ctx context.Context, chatID int64, lID, uID int64, msgID int) {
	err := h.leagueRepo.ApproveMember(ctx, lID, uID)
	if err != nil {
		h.send(chatID, "❌ Тасдиқлашда хатолик юз берди.")
		return
	}
	user, _ := h.userRepo.GetByID(ctx, uID)
	league, _ := h.leagueRepo.GetByID(ctx, lID)

	if user != nil && league != nil {
		playerMsgText := fmt.Sprintf(
			"🎉 <b>Табриклаймиз!</b>\n\nСиз <b>%s</b> лигасига қабул қилиндингиз! ⚽\n\n"+
				"🗓 Ўйинларингиз рўйхатини кўриш учун <b>📅 Чизма</b> тугмасини босинг.\n"+
				"📊 Турнир жадвалини эса <b>📊 Жадвал</b> бўлимида кузатиб боринг.\n\n"+
				"Омад тилаймиз!",
			safe(league.Name),
		)
		playerNotify := tgbotapi.NewMessage(user.TelegramID, playerMsgText)
		playerNotify.ParseMode = "HTML"

		_, err := h.bot.Send(playerNotify)
		if err != nil {
			log.Printf("❌ Ўйинчига хабар юборишда хатолик (ID: %d): %v", user.TelegramID, err)
		}
	}
	h.sendAdminPending(ctx, chatID, msgID)
}
func (h *Handler) adminReject(ctx context.Context, chatID int64, lID, uID int64, msgID int) {
	_ = h.leagueRepo.RejectMember(ctx, lID, uID)

	user, _ := h.userRepo.GetByID(ctx, uID)
	league, _ := h.leagueRepo.GetByID(ctx, lID)

	if user != nil && league != nil {
		rejectMsgText := fmt.Sprintf("😔 Узр, сизнинг <b>%s</b> лигасига юборган аризангиз администратор томонидан рад этилди.", safe(league.Name))

		playerNotify := tgbotapi.NewMessage(user.TelegramID, rejectMsgText)
		playerNotify.ParseMode = "HTML"

		_, err := h.bot.Send(playerNotify)
		if err != nil {
			log.Printf("❌ Рад этиш хабарини юборишда хатолик (ID: %d): %v", user.TelegramID, err)
		}
	}
	h.sendAdminPending(ctx, chatID, msgID)
}
func (h *Handler) handleAdminLeagueNameInput(ctx context.Context, msg *tgbotapi.Message) {
	if !h.isAdmin(ctx, msg.From.ID) {
		return
	}

	name := strings.TrimSpace(msg.Text)
	if len(name) < 2 {
		h.send(msg.Chat.ID, "❗ Ном жуда қисқа. Қайта киритинг:")
		return
	}

	s, err := h.leagueRepo.GetOrCreateActiveSeason(ctx)
	if err != nil {
		h.send(msg.Chat.ID, "❌ Сезон билан хатолик юз берди.")
		return
	}
	_, err = h.leagueRepo.CreateLeague(ctx, s.ID, name, nil, "double", 4, 1, 0)
	if err != nil {
		h.send(msg.Chat.ID, "❌ Лига яратишда хатолик.")
		return
	}

	// Стейтни тозалаймиз
	h.clearState(msg.From.ID)
	success := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ <b>%s</b> лигаси муваффақиятли яратилди!", name))
	success.ParseMode = "HTML"
	h.bot.Send(success)
}
func (h *Handler) notifyPlayersAboutSchedule(ctx context.Context, lID int64, mems []*models.LeagueMember) {
	l, _ := h.leagueRepo.GetByID(ctx, lID)

	for _, m := range mems {
		// Оламиз айнан шу ўйинчининг ўйинларини
		ms, _ := h.matchRepo.GetUserSchedule(ctx, m.UserID, lID)

		// Сарлавҳа: Энди "якунланди" эмас, балки "ўйинлар тақвими яратилди"
		text := fmt.Sprintf("⚔️ <b>%s</b> лигаси учун ўйинлар тақвими яратилди!\n", safe(l.Name))
		text += "Сизнинг ўйинларингиз:\n\n"

		for _, match := range ms {
			var opponentName string
			var matchInfo string

			if match.HomeUserID == m.UserID {
				// Агар Сиз уй эгаси бўлсангиз
				opponentName = safe(match.AwayUser.DisplayName)
				matchInfo = fmt.Sprintf("🏠 <b>%d-тур:</b> Сиз 🆚 %s", match.Round, opponentName)
			} else {
				// Агар Сиз меҳмон бўлсангиз
				opponentName = safe(match.HomeUser.DisplayName)
				matchInfo = fmt.Sprintf("✈️ <b>%d-тур:</b> %s 🆚 Сиз", match.Round, opponentName)
			}
			text += matchInfo + "\n"
		}

		text += "\n⚽ Омад тилаймиз!"

		notify := tgbotapi.NewMessage(m.User.TelegramID, text)
		notify.ParseMode = "HTML"
		h.bot.Send(notify)
	}
}
func (h *Handler) sendAdminPending(ctx context.Context, chatID int64, msgID int) {
	pending, err := h.leagueRepo.GetAllPendingMembers(ctx)
	if err != nil {
		h.send(chatID, "❌ Хатолик: "+err.Error())
		return
	}

	text := "📋 <b>Кутилаётган аризалар:</b>\n\n"
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, p := range pending {
		// Из нашего SQL запроса мы положили название лиги в поле Username для удобства
		leagueName := ""
		if p.User.Username != nil {
			leagueName = *p.User.Username
		}

		text += fmt.Sprintf("🔸 <b>%s</b> (%s)\n", safe(p.User.DisplayName), safe(leagueName))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ "+p.User.DisplayName, fmt.Sprintf("approve_%d_%d", p.LeagueID, p.UserID)),
			tgbotapi.NewInlineKeyboardButtonData("❌", fmt.Sprintf("reject_%d_%d", p.LeagueID, p.UserID)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")))

	if len(pending) == 0 {
		h.smartUpdate(chatID, msgID, MsgNoPending, tgbotapi.NewInlineKeyboardMarkup(rows...))
		return
	}
	h.smartUpdate(chatID, msgID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) sendAdminDisputes(ctx context.Context, chatID int64, msgID int) {
	matches, err := h.matchRepo.GetAllDisputed(ctx)
	if err != nil {
		h.send(chatID, "❌ Хатолик: "+err.Error())
		return
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	if len(matches) == 0 {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
		))
		h.smartUpdate(chatID, msgID, MsgNoDisputes, tgbotapi.NewInlineKeyboardMarkup(rows...))
		return
	}

	text := MsgDisputesHeader
	for _, m := range matches {
		text += fmt.Sprintf("🆔 %d: %s 🆚 %s\n", m.ID, m.HomeUser.DisplayName, m.AwayUser.DisplayName)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("⚖️ %d: %s vs %s", m.ID, m.HomeUser.DisplayName, m.AwayUser.DisplayName),
				fmt.Sprintf("resolve_%d", m.ID),
			),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
	))

	h.smartUpdate(chatID, msgID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) processAdminAdd(ctx context.Context, chatID int64, targetID int64) {
	u, err := h.userRepo.GetByTelegramID(ctx, targetID)
	if err != nil {
		h.send(chatID, fmt.Sprintf("❌ БД хатоси: %v", err))
		return
	}
	if u == nil {
		h.send(chatID, fmt.Sprintf("❌ %d ID топилмади.", targetID))
		return
	}
	h.invalidateAdminCache(targetID)
	h.send(chatID, fmt.Sprintf("✅ <b>%s</b> энди Админ!", safe(u.DisplayName)))
	h.bot.Send(tgbotapi.NewMessage(targetID, "🎉 Сизга Админ ҳуқуқлари берилди!"))
}

func (h *Handler) processAdminRemove(ctx context.Context, chatID int64, targetID int64) {
	u, _ := h.userRepo.GetByTelegramID(ctx, targetID)
	if u == nil {
		h.send(chatID, "❌ Бундай фойдаланувчи топилмади.")
		return
	}
	_ = h.adminRepo.Remove(ctx, u.ID)
	h.invalidateAdminCache(targetID)
	h.send(chatID, fmt.Sprintf("✅ <b>%s</b> админликдан олиб ташланди.", safe(u.DisplayName)))
}

func (h *Handler) sendAdminList(ctx context.Context, chatID int64, msgID int) {
	admins, _ := h.adminRepo.List(ctx)
	text := "👥 <b>Админлар рўйхати:</b>\n\n"
	for _, adm := range admins {

		text += fmt.Sprintf("▫️ %s (<code>%d</code>) — <b>%s</b>\n", safe(adm.User.DisplayName), adm.User.TelegramID, adm.Role)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Қўшиш", "admin_prepare_add"),
			tgbotapi.NewInlineKeyboardButtonData("➖ Ўчириш", "admin_prepare_remove"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
		),
	)

	// Хабарни юборамиз ёки янгилаймиз
	h.smartUpdate(chatID, msgID, text, kb)
}
