package handlers

import (
	"context"
	"efootball-bot/internal/i18n"
	"efootball-bot/internal/models"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) HandleMessage(ctx context.Context, msg *tgbotapi.Message) {
	// Гуруҳда умуман жавоб бермаймиз
	if msg.Chat.Type != "private" {
		return
	}
	text := strings.TrimSpace(msg.Text)
	tgID := msg.From.ID
	chatID := msg.Chat.ID

	if strings.HasPrefix(text, "/") || h.isMenuButton(text) {
		h.clearState(tgID)
	}

	switch text {
	case "/start", MenuMain:
		h.HandleStart(ctx, msg)
		return

	case MenuAdminLeagues:
		h.sendLeagueManageList(ctx, chatID, 0) // lastID эмас, 0 бўлсин!
		return
	case MenuTopScorers:
		h.sendTopScorers(ctx, chatID, 0)
		return
	case MenuStandings:
		h.sendTableSelectorUI(ctx, chatID, 0) // 0
		return
	case MenuHelp:
		h.sendHelp(ctx, chatID, 0)
		return
	case MenuJoinLeague:
		h.sendLeagueList(ctx, chatID, 0) // 0
		return

	case MenuSchedule:
		h.sendScheduleSelectorUI(ctx, chatID, 0, tgID) // 0
		return

	case MenuProfile:
		h.sendProfile(ctx, chatID, 0, tgID) // 0
		return

	case MenuHistory:
		h.sendHistoryLeagueSelect(ctx, chatID, 0, msg.From.ID)
		return

	case MenuResult:
		h.sendMatchesForScore(ctx, chatID, 0, tgID) // lastID
		return

	// --- Админ тугмалари (Ҳаммасига lastID берамиз) ---
	case MenuAdminPending:
		if h.isAdmin(ctx, tgID) {
			h.sendAdminPending(ctx, chatID, 0)
		}
		return

	case MenuAdminDispute:
		if h.isAdmin(ctx, tgID) {
			h.sendAdminDisputes(ctx, chatID, 0)
		}
		return

	case MenuAdminCreate:
		if h.isAdmin(ctx, tgID) {
			h.setState(tgID, StateAdminCreateLeague)
			// Бу ерда ҳам янги хабар юбормай, эскисини таҳрирлаймиз
			kb := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Бекор қилиш", "menu")),
			)
			h.smartUpdate(chatID, 0, "➕ <b>Янги лига номини ёзинг:</b>", kb)
			return
		}

	case MenuAdminManage:
		if h.isAdmin(ctx, tgID) {
			h.sendAdminList(ctx, chatID, 0)
		}
		return

	case MenuAdminDraw:
		if h.isAdmin(ctx, tgID) {
			h.sendLeagueListForDrawUI(ctx, chatID, 0)
		}
		return
	}

	// 2. СТЕЙТЛАР (Ҳолатлар)
	st := h.getState(tgID)
	switch st {
	case StateWaitingName:
		h.handleNameInput(ctx, msg)
	case StateAdminCreateLeague:
		h.handleAdminLeagueNameInput(ctx, msg)
	case StateWaitingNewName:
		h.handleNewNameChange(ctx, msg)

	case StateAdminWaitingAddID:
		id, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			h.send(chatID, "❌ <b>Хато:</b> Фақат рақам (ID) ёзинг.")
			return
		}
		h.processAdminAdd(ctx, chatID, id)
		h.clearState(tgID)

	case StateAdminWaitingRemoveID:
		id, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			h.send(chatID, "❌ <b>Хато:</b> Фақат рақам (ID) ёзинг.")
			return
		}
		h.processAdminRemove(ctx, chatID, id)
		h.clearState(tgID)
	case StateWaitingTeamPower:
		h.handleTeamPowerInput(ctx, msg, false)

	case StateWaitingNewTeamPower:
		h.handleTeamPowerInput(ctx, msg, true)

	default:
		// Агар тушунарсиз матн келса, эски менюни ўчириб, янги хабар чиқариши мумкин
		h.send(chatID, "🤔 Тушунмадим. Пастдаги менюдан фойдаланинг.")
	}
}
func (h *Handler) handleNameInput(ctx context.Context, msg *tgbotapi.Message) {
	name := strings.TrimSpace(msg.Text)
	if len(name) < 2 {
		m := tgbotapi.NewMessage(msg.Chat.ID, MsgNameTooShort)
		m.ParseMode = "HTML"
		h.bot.Send(m)
		return
	}
	if len(name) > 64 {
		m := tgbotapi.NewMessage(msg.Chat.ID, MsgNameTooLong)
		m.ParseMode = "HTML"
		h.bot.Send(m)
		return
	}

	// Яхши ном — Базага ёзамиз!
	var username *string
	if msg.From.UserName != "" {
		u := msg.From.UserName
		username = &u
	}

	newUser, err := h.userRepo.Create(ctx, msg.From.ID, name, username)
	if err == nil && newUser != nil {
		// Тилни олиб сақлаймиз
		if langVal, ok := h.states.Load(fmt.Sprintf("lang_%d", msg.From.ID)); ok {
			lang := langVal.(string)
			_ = h.userRepo.UpdateLanguage(ctx, newUser.ID, lang)
			h.states.Delete(fmt.Sprintf("lang_%d", msg.From.ID))
		}
	}
	if err != nil {
		h.send(msg.Chat.ID, "❌ Кечирасиз, база билан боғланишда хато чиқди. Қайтадан синаб кўринг.")
		return
	}

	h.setState(msg.From.ID, StateWaitingTeamPower)
	tpMsg := tgbotapi.NewMessage(msg.Chat.ID,
		fmt.Sprintf("✅ Зўр, <b>%s</b>!\n\n"+MsgTeamPowerAsk, name))
	tpMsg.ParseMode = "HTML"
	h.bot.Send(tpMsg)
}
func (h *Handler) handleNewNameChange(ctx context.Context, msg *tgbotapi.Message) {
	name := strings.TrimSpace(msg.Text)
	telegramID := msg.From.ID
	chatID := msg.Chat.ID

	if len(name) < 2 {
		h.send(chatID, "❗ Исм жуда қисқа. Камида 2 та белги бўлиши керак.")
		return
	}
	if len(name) > 32 {
		h.send(chatID, "❗ Исм жуда узун. Максимум 32 та белги.")
		return
	}

	user, err := h.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil || user == nil {
		h.send(chatID, "❌ Фойдаланувчи топилмади.")
		return
	}

	err = h.userRepo.UpdateDisplayName(ctx, user.ID, name)
	if err != nil {
		h.send(chatID, "❌ Базада исмни янгилашда хатолик юз берди.")
		return
	}
	h.clearState(telegramID)

	h.send(chatID, fmt.Sprintf("✅ Исм ўзгартирилди: <b>%s</b>", name))

	h.sendProfile(ctx, chatID, 0, telegramID)
}
func (h *Handler) sendProfile(ctx context.Context, chatID int64, msgID int, telegramID int64) {
	u, err := h.userRepo.GetByTelegramID(ctx, telegramID)
	if err != nil || u == nil {
		h.send(chatID, i18n.T("uz", "not.registered"))
		return
	}

	lang := u.Language
	if lang == "" {
		lang = "uz"
	}

	text := fmt.Sprintf(i18n.T(lang, "profile.title")+"\n", safe(u.DisplayName))
	text += fmt.Sprintf(i18n.T(lang, "profile.date")+"\n", u.CreatedAt.Format("02.01.2006"))
	text += fmt.Sprintf(i18n.T(lang, "profile.id")+"\n\n", u.TelegramID)
	text += fmt.Sprintf(i18n.T(lang, "profile.rating")+"\n", u.Rating)
	text += fmt.Sprintf(i18n.T(lang, "profile.rank")+"\n", u.Rank)

	if u.TeamPower > 0 {
		text += fmt.Sprintf(i18n.T(lang, "profile.teampower")+"\n", u.TeamPower)
	} else {
		text += i18n.T(lang, "profile.teampower.empty") + "\n"
	}

	text += "\n" + i18n.T(lang, "profile.stats") + "\n"

	memberships, _ := h.leagueRepo.GetUserLeagues(ctx, u.ID)

	var nextMatch *models.Match

	if len(memberships) == 0 {
		text += i18n.T(lang, "profile.nostats")
	} else {
		for _, m := range memberships {
			text += fmt.Sprintf("🏆 <b>%s:</b> %d очко | %d ўйин\n",
				safe(m.League.Name), m.Points, m.Wins+m.Draws+m.Losses)
			if nextMatch == nil {
				ms, _ := h.matchRepo.GetUserSchedule(ctx, u.ID, m.LeagueID)
				for _, match := range ms {
					if match.HomeUserID == u.ID {
						nextMatch = match
						break
					}
				}
			}
		}
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	if nextMatch != nil {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("📝 %d-тур натижасини киритиш", nextMatch.Round),
				fmt.Sprintf("enter_match_%d", nextMatch.ID),
			),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "btn.editname"), "edit_name"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "btn.editteampower"), "edit_team_power"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🌐 "+i18n.T(lang, "menu.lang"), "change_lang"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "btn.back"), "menu"),
	))

	h.smartUpdate(chatID, msgID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
func (h *Handler) handleTeamPowerInput(ctx context.Context, msg *tgbotapi.Message, isUpdate bool) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID
	tgID := msg.From.ID

	tp, err := strconv.Atoi(text)
	if err != nil || tp < 0 {
		h.send(chatID, "❗ Фақат мусбат рақам киритинг. Мисол: <b>3250</b>")
		return
	}

	user, _ := h.userRepo.GetByTelegramID(ctx, tgID)
	if user == nil {
		h.send(chatID, MsgNotRegistered)
		return
	}

	_ = h.userRepo.UpdateTeamPower(ctx, user.ID, tp)
	h.clearState(tgID)

	lang := "uz"
	if user.Language != "" {
		lang = user.Language
	}

	if isUpdate {
		h.send(chatID, fmt.Sprintf("✅ %s: <b>%d</b>", i18n.T(lang, "btn.editteampower"), tp))
		h.sendProfile(ctx, chatID, 0, tgID)
	} else {
		bottomKB := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton(i18n.T(lang, "menu.main")),
				tgbotapi.NewKeyboardButton(i18n.T(lang, "menu.profile")),
			),
		)
		bottomKB.ResizeKeyboard = true
		doneMsg := tgbotapi.NewMessage(chatID,
			fmt.Sprintf(i18n.T(lang, "register.done"), safe(user.DisplayName), tp))
		doneMsg.ParseMode = "HTML"
		doneMsg.ReplyMarkup = bottomKB
		h.bot.Send(doneMsg)
		h.sendMenuWithID(ctx, chatID, 0, tgID, nil)
	}
}
func (h *Handler) sendHelp(ctx context.Context, chatID int64, msgID int) {
	h.smartUpdate(chatID, msgID, MsgHelp, tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
		),
	))
}
