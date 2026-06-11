package handlers

import (
	"context"
	"fmt"
	"time"

	"strings"
	"sync"

	"efootball-bot/internal/i18n"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type BotAPI interface {
	Send(tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

type State string

const (
	StateNone                 State = ""
	StateWaitingName          State = "waiting_name"
	StateWaitingNewName       State = "waiting_new_name"
	StateWaitingTeamPower     State = "waiting_team_power"
	StateWaitingNewTeamPower  State = "waiting_new_team_power"
	StateAdminCreateLeague    State = "admin_create_league"
	StateAdminWaitingAddID    State = "admin_waiting_add_id"
	StateAdminWaitingRemoveID State = "admin_waiting_remove_id"
	StateWaitingLang          State = "waiting_lang"
)

type Handler struct {
	bot           BotAPI
	userRepo      repository.UserRepository
	leagueRepo    repository.LeagueRepository
	matchRepo     repository.MatchRepository
	matchSvc      *service.MatchService
	schedSvc      *service.ScheduleService
	groupStageSvc *service.GroupStageService
	eloSvc        *service.EloService
	adminRepo     repository.AdminRepository
	achievRepo    repository.AchievementRepository
	adminID       int64
	groupID       int64
	states        sync.Map
	menuMsgID     sync.Map
	adminCache    sync.Map
	adminTTL      time.Duration
}

func (h *Handler) SetAchievementRepo(a repository.AchievementRepository) { h.achievRepo = a }

func New(bot BotAPI, userRepo repository.UserRepository, leagueRepo repository.LeagueRepository, matchRepo repository.MatchRepository, matchSvc *service.MatchService, schedSvc *service.ScheduleService, groupStageSvc *service.GroupStageService, adminRepo repository.AdminRepository, eloSvc *service.EloService, adminID int64, groupID int64) *Handler {
	return &Handler{
		bot:           bot,
		userRepo:      userRepo,
		leagueRepo:    leagueRepo,
		matchRepo:     matchRepo,
		matchSvc:      matchSvc,
		schedSvc:      schedSvc,
		groupStageSvc: groupStageSvc,
		eloSvc:        eloSvc,
		adminRepo:     adminRepo,
		adminID:       adminID,
		groupID:       groupID,
		states:        sync.Map{},
		adminTTL:      5 * time.Minute,
	}
}

func (h *Handler) setState(id int64, s State) { h.states.Store(id, s) }
func (h *Handler) getState(id int64) State {
	val, ok := h.states.Load(id)
	if ok {
		return val.(State)
	}
	return StateNone
}
func (h *Handler) clearState(id int64) { h.states.Delete(id) }

func (h *Handler) isAdmin(ctx context.Context, id int64) bool {
	// Главный админ всегда админ — без запросов
	if id == h.adminID {
		return true
	}

	// Проверяем кеш
	if val, ok := h.adminCache.Load(id); ok {
		cached := val.(time.Time)
		if time.Since(cached) < h.adminTTL {
			return true
		}
		// Кеш устарел — удаляем
		h.adminCache.Delete(id)
	}

	// Идём в БД
	is, err := h.adminRepo.IsAdmin(ctx, id)
	if err != nil {
		return false
	}

	// Сохраняем в кеш только если реально админ
	if is {
		h.adminCache.Store(id, time.Now())
	}

	return is
}
func (h *Handler) invalidateAdminCache(id int64) {
	h.adminCache.Delete(id)
}
func (h *Handler) HandleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) bool {
	if cb.Message.Chat.Type != "private" {
		return false
	}
	h.bot.Request(tgbotapi.NewCallback(cb.ID, ""))
	data := cb.Data
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID
	telegramID := cb.From.ID

	switch {
	case data == "menu":
		h.bot.Request(tgbotapi.NewDeleteMessage(chatID, msgID))
		user, _ := h.userRepo.GetByTelegramID(ctx, telegramID)
		name := "Ўйинчи"
		if user != nil {
			name = user.DisplayName
		}
		var welcomeText string
		if h.isAdmin(ctx, telegramID) {
			welcomeText = fmt.Sprintf("👑 <b>Админ панели</b>\nСалом, <b>%s</b>!\n\nПастдаги тугмалардан фойдаланинг.", safe(name))
		} else {
			welcomeText = fmt.Sprintf("✅ <b>%s</b>, пастдаги менюдан фойдаланинг!", safe(name))
		}
		m := tgbotapi.NewMessage(chatID, welcomeText)
		m.ParseMode = "HTML"
		h.bot.Send(m)
	case data == "lang_uz", data == "lang_ru", data == "lang_tg":
		lang := strings.TrimPrefix(data, "lang_")
		user, _ := h.userRepo.GetByTelegramID(ctx, telegramID)

		if user == nil {
			// Янги фойдаланувчи — тилни сақлаб регистрацияга ўтамиз
			// Вақтинча тилни state да сақлаймиз
			h.states.Store(fmt.Sprintf("lang_%d", telegramID), lang)
			h.setState(telegramID, StateWaitingName)
			welcomeText := i18n.T(lang, "welcome")
			m := tgbotapi.NewMessage(chatID, welcomeText)
			m.ParseMode = "HTML"
			m.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			h.bot.Request(tgbotapi.NewDeleteMessage(chatID, msgID))
			h.bot.Send(m)
		} else {
			// Мавжуд фойдаланувчи — тилни янгилаймиз
			_ = h.userRepo.UpdateLanguage(ctx, user.ID, lang)
			changed := i18n.T(lang, "lang.changed")
			h.smartUpdate(chatID, msgID, changed, tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "btn.back"), "menu"),
				),
			))
		}
	case data == "join_league":
		h.sendLeagueList(ctx, chatID, msgID)

	case strings.HasPrefix(data, "join_"):
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "join_"), "%d", &lID)
		h.joinLeagueByID(ctx, chatID, telegramID, lID, msgID)

	case data == "table":
		h.sendTableSelectorUI(ctx, chatID, msgID)
	case data == "change_lang":
		h.sendLangSelect(chatID, msgID)
	case data == "top_scorers":
		h.sendTopScorers(ctx, chatID, msgID)

	case strings.HasPrefix(data, "table_"):
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "table_"), "%d", &lID)
		h.sendTableUI(ctx, chatID, msgID, lID)

	case data == "help":
		h.smartUpdate(chatID, msgID, MsgHelp, tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
			),
		))

	case strings.HasPrefix(data, "history_league_"):
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "history_league_"), "%d", &lID)
		h.sendMatchHistoryByLeague(ctx, chatID, msgID, telegramID, lID)

	case strings.HasPrefix(data, "sched_view_"):
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "sched_view_"), "%d", &lID)
		h.sendScheduleByLeagueUI(ctx, chatID, msgID, telegramID, lID)

	case data == "admin_reset_ratings_confirm":
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Ҳа, сброс қиламан", "admin_reset_ratings_do"),
				tgbotapi.NewInlineKeyboardButtonData("❌ Бекор", "admin_manage_admins"),
			),
		)
		h.smartUpdate(chatID, msgID,
			"⚠️ <b>Диққат!</b>\n\nБарча ўйинчиларнинг рейтинги <b>1000</b> га қайтарилади!\n\nДавом этасизми?", kb)

	case data == "admin_reset_ratings_do":
		err := h.userRepo.ResetAllRatings(ctx)
		if err != nil {
			h.smartUpdate(chatID, msgID, "❌ Хатолик: "+err.Error(),
				tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(BtnBack, "admin_manage_admins"),
					),
				))
			return false
		}
		h.smartUpdate(chatID, msgID,
			"✅ <b>Барча ўйинчиларнинг рейтинги сброс қилинди!</b>\n\nҲамма 1000 дан бошлайди. 🔄",
			tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(BtnBack, "admin_manage_admins"),
				),
			))

	case data == "schedule":
		h.sendSchedule(ctx, chatID, msgID, telegramID)

	case data == "profile":
		h.sendProfile(ctx, chatID, msgID, telegramID)

	case data == "match_history":
		h.sendHistoryLeagueSelect(ctx, chatID, msgID, telegramID)

	case data == "enter_result_list":
		h.sendMatchesForScore(ctx, chatID, msgID, telegramID)

	case strings.HasPrefix(data, "enter_match_"):
		var mID int64
		fmt.Sscanf(strings.TrimPrefix(data, "enter_match_"), "%d", &mID)
		h.editScoreMessage(ctx, chatID, msgID, telegramID, mID, 0, 0)

	case strings.HasPrefix(data, "s_mod_"):
		var mID int64
		var hg, ag int16
		fmt.Sscanf(strings.TrimPrefix(data, "s_mod_"), "%d_%d_%d", &mID, &hg, &ag)
		if hg < 0 || ag < 0 {
			return false
		}
		h.editScoreMessage(ctx, chatID, msgID, telegramID, mID, hg, ag)

	case strings.HasPrefix(data, "s_sub_"):
		var mID int64
		var hg, ag int16
		fmt.Sscanf(strings.TrimPrefix(data, "s_sub_"), "%d_%d_%d", &mID, &hg, &ag)
		lockKey := fmt.Sprintf("submit_%d", mID)
		if _, loaded := h.states.LoadOrStore(lockKey, true); loaded {
			return false
		}
		defer h.states.Delete(lockKey)
		match, err := h.matchSvc.ClaimResult(ctx, mID, cb.From.ID, hg, ag)
		if err != nil {
			h.send(chatID, "❌ Хатолик: "+err.Error())
			return false
		}
		aU, _ := h.userRepo.GetByID(ctx, match.AwayUserID)
		if aU != nil {
			kb := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(BtnAgree, fmt.Sprintf("confirm_%d", mID)),
					tgbotapi.NewInlineKeyboardButtonData(BtnDisagree, fmt.Sprintf("dispute_%d", mID)),
				),
			)
			text := fmt.Sprintf("⚽ <b>%s</b> натижа киритди: <b>%d-%d</b>. Тасдиқлайсизми?",
				safe(match.HomeUser.DisplayName), hg, ag)
			notify := tgbotapi.NewMessage(aU.TelegramID, text)
			notify.ParseMode = "HTML"
			notify.ReplyMarkup = kb
			h.bot.Send(notify)
		}
		h.send(chatID, MsgResultSent)

	case data == "admin_pending":
		if h.isAdmin(ctx, telegramID) {
			h.sendAdminPending(ctx, chatID, msgID)
		}

	case data == "admin_disputes":
		if h.isAdmin(ctx, telegramID) {
			h.sendAdminDisputes(ctx, chatID, msgID)
		}

	case data == "admin_draw":
		if h.isAdmin(ctx, telegramID) {
			h.sendLeagueListForDrawUI(ctx, chatID, msgID)
		}

	case data == "edit_name":
		h.setState(telegramID, StateWaitingNewName)
		h.smartUpdate(chatID, msgID, "🆕 <b>Янги исмингизни киритинг:</b>",
			tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("❌ Бекор қилиш", "profile"),
				),
			))

	case data == "edit_team_power":
		h.setState(telegramID, StateWaitingNewTeamPower)
		h.smartUpdate(chatID, msgID,
			"⚡ <b>Янги Команда кучингизни киритинг:</b>\n\n"+
				"📊 eFootball → Мой клуб → Общая сила\n\n"+
				"<i>Мисол: 3250</i>",
			tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("❌ Бекор қилиш", "profile"),
				),
			))

	case strings.HasPrefix(data, "draw_"):
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "draw_"), "%d", &lID)
		h.adminDraw(ctx, chatID, lID, msgID)

	case data == "admin_manage_admins":
		h.sendAdminManageSubMenu(ctx, chatID, msgID)

	case data == "admin_prepare_add":
		h.setState(telegramID, StateAdminWaitingAddID)
		h.smartUpdate(chatID, msgID,
			"🆔 <b>Янги админнинг Telegram ID рақамини ёзинг:</b>\n\n(Бу рақамни ўша фойдаланувчи Профил бўлимидан кўриши мумкин)",
			tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "admin_manage_admins"),
				),
			))

	case data == "admin_prepare_remove":
		h.setState(telegramID, StateAdminWaitingRemoveID)
		h.smartUpdate(chatID, msgID, "🆔 <b>Ўчириладиган админнинг Telegram ID рақамини ёзинг:</b>",
			tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "admin_manage_admins"),
				),
			))

	case data == "admin_create_league":
		if h.isAdmin(ctx, telegramID) {
			h.setState(telegramID, StateAdminCreateLeague)
			h.smartUpdate(chatID, msgID, "🏆 <b>Янги лига номини киритинг:</b>",
				tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(BtnCancel, "menu"),
					),
				))
		}

	case strings.HasPrefix(data, "approve_"):
		if !h.isAdmin(ctx, telegramID) {
			return false
		}
		var lID, uID int64
		fmt.Sscanf(strings.TrimPrefix(data, "approve_"), "%d_%d", &lID, &uID)
		h.adminApprove(ctx, chatID, lID, uID, msgID)

	case strings.HasPrefix(data, "reject_"):
		if !h.isAdmin(ctx, telegramID) {
			return false
		}
		var lID, uID int64
		fmt.Sscanf(strings.TrimPrefix(data, "reject_"), "%d_%d", &lID, &uID)
		h.adminReject(ctx, chatID, lID, uID, msgID)

	case strings.HasPrefix(data, "resolve_"):
		if !h.isAdmin(ctx, telegramID) {
			return false
		}
		var mID int64
		var hg, ag int16
		if _, err := fmt.Sscanf(strings.TrimPrefix(data, "resolve_"), "%d_%d_%d", &mID, &hg, &ag); err == nil {
			h.adminResolveMatch(ctx, chatID, mID, hg, ag)
			h.sendAdminDisputes(ctx, chatID, msgID)
		}

	case strings.HasPrefix(data, "confirm_"):
		var mID int64
		fmt.Sscanf(strings.TrimPrefix(data, "confirm_"), "%d", &mID)
		h.confirmMatch(ctx, chatID, mID)
		h.sendMenuWithID(ctx, chatID, msgID, telegramID, nil)

	case strings.HasPrefix(data, "dispute_"):
		var mID int64
		fmt.Sscanf(strings.TrimPrefix(data, "dispute_"), "%d", &mID)
		h.disputeMatch(ctx, chatID, mID)
		h.sendMenuWithID(ctx, chatID, msgID, telegramID, nil)
	}

	return false
}
func (h *Handler) HandleStart(ctx context.Context, msg *tgbotapi.Message) {
	// Гуруҳда умуман жавоб бермаймиз
	if msg.Chat.Type != "private" {
		return
	}
	h.clearState(msg.From.ID)
	user, _ := h.userRepo.GetByTelegramID(ctx, msg.From.ID)

	if user == nil {
		h.setState(msg.From.ID, StateWaitingLang)
		h.sendLangSelect(msg.Chat.ID, 0)
		return
	}

	var buttons [][]tgbotapi.KeyboardButton

	if h.isAdmin(ctx, msg.From.ID) {
		buttons = [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton(MenuMain), tgbotapi.NewKeyboardButton(MenuProfile)},
			{tgbotapi.NewKeyboardButton(MenuJoinLeague), tgbotapi.NewKeyboardButton(MenuStandings)},
			{tgbotapi.NewKeyboardButton(MenuSchedule), tgbotapi.NewKeyboardButton(MenuHistory)},
			{tgbotapi.NewKeyboardButton(MenuTopScorers), tgbotapi.NewKeyboardButton(MenuResult)},
			{tgbotapi.NewKeyboardButton(MenuAdminCreate), tgbotapi.NewKeyboardButton(MenuAdminLeagues)},
			{tgbotapi.NewKeyboardButton(MenuAdminPending), tgbotapi.NewKeyboardButton(MenuAdminDraw)},
			{tgbotapi.NewKeyboardButton(MenuAdminDispute), tgbotapi.NewKeyboardButton(MenuAdminManage)},
			{tgbotapi.NewKeyboardButton(MenuHelp)},
		}
	} else {
		buttons = [][]tgbotapi.KeyboardButton{
			{tgbotapi.NewKeyboardButton(MenuJoinLeague)},
			{tgbotapi.NewKeyboardButton(MenuStandings), tgbotapi.NewKeyboardButton(MenuSchedule)},
			{tgbotapi.NewKeyboardButton(MenuResult), tgbotapi.NewKeyboardButton(MenuHistory)},
			{tgbotapi.NewKeyboardButton(MenuTopScorers), tgbotapi.NewKeyboardButton(MenuProfile)},
			{tgbotapi.NewKeyboardButton(MenuHelp)},
		}
	}

	keyboard := tgbotapi.NewReplyKeyboard(buttons...)
	keyboard.ResizeKeyboard = true

	// Скриншотингиздаги иккита хабар чиқмаслиги учун шу ерни ўзидаёқ саломлашамиз
	welcomeText := fmt.Sprintf("👑 <b>Админ панели</b>\nСалом, <b>%s</b>!\n\nБошқарув учун пастдаги тугмалардан фойдаланинг.", safe(user.DisplayName))
	if !h.isAdmin(ctx, msg.From.ID) {
		welcomeText = fmt.Sprintf("✅ <b>%s</b>, асосий менюга хуш келибсиз!", safe(user.DisplayName))
	}

	m := tgbotapi.NewMessage(msg.Chat.ID, welcomeText)
	m.ParseMode = "HTML"
	m.ReplyMarkup = keyboard
	h.bot.Send(m)

}

func (h *Handler) sendMenuWithID(ctx context.Context, chatID int64, msgID int, telegramID int64, bottomMenu interface{}) {
	user, _ := h.userRepo.GetByTelegramID(ctx, telegramID)
	name := "Ўйинчи"
	if user != nil {
		name = user.DisplayName
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	var msgBody string

	if h.isAdmin(ctx, telegramID) {
		msgBody = fmt.Sprintf(MsgAdminMenu, safe(name))
		rows = [][]tgbotapi.InlineKeyboardButton{
			{tgbotapi.NewInlineKeyboardButtonData("🏆 Лигага қўшилиш", "join_league")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnTable, "table"), tgbotapi.NewInlineKeyboardButtonData(BtnSchedule, "schedule")},
			{tgbotapi.NewInlineKeyboardButtonData("👤 Профил", "profile"), tgbotapi.NewInlineKeyboardButtonData("📜 Тарих", "match_history")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnCreateLeague, "admin_create_league"), tgbotapi.NewInlineKeyboardButtonData(BtnManageLeagues, "admin_manage_leagues")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnPending, "admin_pending"), tgbotapi.NewInlineKeyboardButtonData(BtnDraw, "admin_draw")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnTopScorers, "top_scorers")},
			{tgbotapi.NewInlineKeyboardButtonData("⚖️ Низолар", "admin_disputes"), tgbotapi.NewInlineKeyboardButtonData("👥 Админлар", "admin_manage_admins")},
		}
	} else {
		msgBody = fmt.Sprintf("✅ <b>%s</b>, асосий менюга хуш келибсиз!", safe(name))
		rows = [][]tgbotapi.InlineKeyboardButton{
			{tgbotapi.NewInlineKeyboardButtonData(BtnJoinLeague, "join_league")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnTable, "table"), tgbotapi.NewInlineKeyboardButtonData(BtnSchedule, "schedule")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnEnterResult, "enter_result_list")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnTopScorers, "top_scorers")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnProfile, "profile")},
			{tgbotapi.NewInlineKeyboardButtonData(BtnHelp, "help")},
		}
	}

	h.smartUpdate(chatID, msgID, msgBody, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) sendLeagueManageList(ctx context.Context, chatID int64, msgID int) {
	leagues, _ := h.leagueRepo.GetAllLeagues(ctx)
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, l := range leagues {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ "+l.Name, fmt.Sprintf("league_manage_%d", l.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")))
	h.smartUpdate(chatID, msgID, MsgChooseLeagueManage, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) adminResolveMatch(ctx context.Context, chatID int64, mID int64, hg, ag int16) {
	if _, err := h.matchSvc.AdminResolve(ctx, mID, hg, ag, h.adminID, "Решено"); err != nil {
		h.send(chatID, "❌ Хатолик: "+err.Error())
	}
}

func (h *Handler) AdminResolve(ctx context.Context, msg *tgbotapi.Message, mID int64, hg, ag int16) {
	h.adminResolveMatch(ctx, msg.Chat.ID, mID, hg, ag)
}

func (h *Handler) Send(chatID int64, text string) { h.send(chatID, text) }

// adminApprove — одобряет игрока и обновляет список заявок (в том же окне)

// sendLeagueListForDrawUI — выбор лиги для жеребьевки
func (h *Handler) sendLeagueListForDrawUI(ctx context.Context, chatID int64, msgID int) {
	ls, _ := h.leagueRepo.GetActiveLeagues(ctx)
	var rows [][]tgbotapi.InlineKeyboardButton
	found := false
	for _, l := range ls {
		if l.Status == models.LeagueRegistration {
			found = true
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⚔️ Қуръа: "+l.Name, fmt.Sprintf("draw_%d", l.ID)),
			))
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu")))...)

	text := "⚔️ Қуръа ташлаш учун лигани танланг:\n_(Фақат рўйхатга олиш кетаётган лигалар)_"
	if !found {
		text = "❌ Ҳозирда қуръа ташланадиган (янги) лигалар йўқ."
	}

	h.smartUpdate(chatID, msgID, text, kb)
}

func (a *AdminHandlers) sendLeagueMembers(ctx context.Context, chatID int64, lID int64, msgID int) {
	members, err := a.leagueRepo.GetMembers(ctx, lID)
	league, _ := a.leagueRepo.GetByID(ctx, lID)

	if err != nil || len(members) == 0 {
		a.smartUpdate(chatID, msgID, "😔 Бу лигада ҳали иштирокчилар йўқ.", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "admin_manage_leagues")),
		))
		return
	}

	text := fmt.Sprintf("👥 <b>%s — иштирокчилари:</b>\n\n", safe(league.Name))
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, m := range members {
		uName := "User"
		if m.User != nil {
			uName = m.User.DisplayName
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 "+uName+" ни чиқариш", fmt.Sprintf("kick_%d_%d", lID, m.UserID)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, fmt.Sprintf("league_manage_%d", lID))))

	a.smartUpdate(chatID, msgID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (h *Handler) sendAdminManageSubMenu(ctx context.Context, chatID int64, msgID int) {
	text := "👥 <b>Админларни бошқариш панели</b>\n\nНима қилмоқчисиз?"
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Рўйхатни кўриш", "admin_list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕ Қўшиш", "admin_prepare_add"),
			tgbotapi.NewInlineKeyboardButtonData("➖ Ўчириш", "admin_prepare_remove"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnResetRatings, "admin_reset_ratings_confirm"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
		),
	)
	h.smartUpdate(chatID, msgID, text, kb)
}
func (h *Handler) sendLangSelect(chatID int64, msgID int) {
	text := "🌐 Тилни танланг / Выберите язык / Забонро интихоб кунед:"
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇺🇿 Узбекча", "lang_uz"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇷🇺 Русский", "lang_ru"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇹🇯 Тоҷикӣ", "lang_tg"),
		),
	)
	if msgID > 0 {
		h.smartUpdate(chatID, msgID, text, kb)
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = kb
		h.bot.Send(msg)
	}
}
