package handlers

import (
	"context"
	"fmt"
	"strings"

	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AdminHandlers struct {
	bot        BotAPI
	userRepo   repository.UserRepository
	leagueRepo repository.LeagueRepository
	adminRepo  repository.AdminRepository
	adminID    int64
}

func NewAdminHandlers(bot BotAPI, userRepo repository.UserRepository, leagueRepo repository.LeagueRepository, adminRepo repository.AdminRepository, superAdminID int64) *AdminHandlers {
	return &AdminHandlers{bot: bot, userRepo: userRepo, leagueRepo: leagueRepo, adminRepo: adminRepo, adminID: superAdminID}
}

func (a *AdminHandlers) smartUpdate(chatID int64, msgID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	if msgID > 0 {
		edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, msgID, text, keyboard)
		edit.ParseMode = "HTML"
		a.bot.Request(edit)
	} else {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		a.bot.Send(msg)
	}
}

func (a *AdminHandlers) HandleCallback(ctx context.Context, cb *tgbotapi.CallbackQuery) bool {
	a.bot.Request(tgbotapi.NewCallback(cb.ID, ""))
	data := cb.Data
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID
	telegramID := cb.From.ID

	isSA, _ := a.adminRepo.IsSuperAdmin(ctx, telegramID)
	if telegramID == a.adminID {
		isSA = true
	}

	switch {
	case data == "admin_manage_leagues":
		if !isSA {
			a.send(chatID, MsgNotSuperAdmin)
			return true
		}
		a.sendLeagueManageList(ctx, chatID, msgID)
		return true
	case strings.HasPrefix(data, "league_archive_"):
		if !isSA {
			a.send(chatID, MsgNotSuperAdmin)
			return true
		}
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "league_archive_"), "%d", &lID)
		_ = a.leagueRepo.ArchiveLeague(ctx, lID)
		a.sendLeagueManageList(ctx, chatID, msgID)
		return true

	case strings.HasPrefix(data, "league_confirm_del_"):
		if !isSA {
			a.send(chatID, MsgNotSuperAdmin)
			return true
		}
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "league_confirm_del_"), "%d", &lID)

		text := "⚠️ <b>Диққат! Лигани ўчирсангиз, барча ўйинлар ва статистикалар бутунлай йўқолади.</b>"
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🗑 Ҳа, ўчирилсин", fmt.Sprintf("league_delete_%d", lID))),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "admin_manage_leagues")),
		)
		a.smartUpdate(chatID, msgID, text, kb)
		return true

	case strings.HasPrefix(data, "league_delete_"):
		if !isSA {
			a.send(chatID, MsgNotSuperAdmin)
			return true
		}
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "league_delete_"), "%d", &lID)
		_ = a.leagueRepo.DeleteLeague(ctx, lID)
		a.sendLeagueManageList(ctx, chatID, msgID)
		return true

	case strings.HasPrefix(data, "league_manage_"):
		if !isSA {
			a.send(chatID, MsgNotSuperAdmin)
			return true
		}
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "league_manage_"), "%d", &lID)
		a.sendLeagueActions(ctx, chatID, msgID, lID)
		return true

	case strings.HasPrefix(data, "view_mems_"):
		if !isSA {
			a.send(chatID, MsgNotSuperAdmin)
			return true
		}
		var lID int64
		fmt.Sscanf(strings.TrimPrefix(data, "view_mems_"), "%d", &lID)
		a.sendLeagueMembers(ctx, chatID, lID, msgID)
		return true

	case strings.HasPrefix(data, "kick_"):
		if !isSA {
			a.send(chatID, MsgNotSuperAdmin)
			return true
		}
		var lID, uID int64
		fmt.Sscanf(strings.TrimPrefix(data, "kick_"), "%d_%d", &lID, &uID)
		league, _ := a.leagueRepo.GetByID(ctx, lID)
		if league != nil && league.Status != models.LeagueRegistration {

			a.bot.Send(tgbotapi.NewMessage(chatID, "❌ <b>Хато:</b> Қуръа ташлангандан кейин иштирокчини ўчириш мумкин эмас! Бу ўйинлар жадвалини бузиб юборади."))
			return true
		}
		_ = a.leagueRepo.RemoveMember(ctx, lID, uID)
		a.sendLeagueMembers(ctx, chatID, lID, msgID)
		return true

	default:
		return false
	}
}
func (a *AdminHandlers) sendLeagueActions(ctx context.Context, chatID int64, msgID int, lID int64) {
	league, err := a.leagueRepo.GetByID(ctx, lID)
	if err != nil || league == nil {
		a.send(chatID, MsgLeagueNotFound)
		return
	}
	text := fmt.Sprintf("⚙️ <b>Лига:</b> %s\n🏷 <b>Ҳолати:</b> %s", safe(league.Name), league.Status)

	var rows [][]tgbotapi.InlineKeyboardButton

	if league.Status != models.LeagueArchived {
		// Қаторларга бўлиб чиқарамиз (чиройлироқ кўринади)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("👥 Иштирокчилар", fmt.Sprintf("view_mems_%d", lID))))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📦 Архивга суриш", fmt.Sprintf("league_archive_%d", lID))))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🗑 Бутунлай ўчириш", fmt.Sprintf("league_confirm_del_%d", lID))))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(BtnBack, "admin_manage_leagues")))

	a.smartUpdate(chatID, msgID, text, tgbotapi.NewInlineKeyboardMarkup(rows...))
}

func (a *AdminHandlers) send(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = "HTML"
	a.bot.Send(m)
}

func (a *AdminHandlers) AddAdmin(ctx context.Context, chatID int64, callerID int64, targetTelegramID int64, role models.AdminRole) {
	isSA, _ := a.adminRepo.IsSuperAdmin(ctx, callerID)
	if callerID != a.adminID && !isSA {
		a.send(chatID, "❌ Бу амал фақат бош admin учун.")
		return
	}
	u, _ := a.userRepo.GetByTelegramID(ctx, targetTelegramID)
	if u == nil {
		a.send(chatID, "❌ Фойдаланувчи топилмади. У аввал ботга кириб /start босиши керак.")
		return
	}
	err := a.adminRepo.Add(ctx, u.ID, role)
	if err != nil {
		a.send(chatID, "❌ Хатолик юз берди.")
		return
	}

	a.send(chatID, fmt.Sprintf("✅ <b>%s</b> энди ботда <b>%s</b> ролига эга!", safe(u.DisplayName), role))
	a.send(targetTelegramID, fmt.Sprintf("🎉 Сизга <b>%s</b> ҳуқуқлари берилди!", role))
}

func (a *AdminHandlers) RemoveAdmin(ctx context.Context, chatID int64, callerID int64, targetTelegramID int64) {
	isSA, _ := a.adminRepo.IsSuperAdmin(ctx, callerID)
	if callerID != a.adminID && !isSA {
		a.send(chatID, MsgNotSuperAdmin)
		return
	}

	u, _ := a.userRepo.GetByTelegramID(ctx, targetTelegramID)
	if u == nil {
		a.send(chatID, MsgUserNotFound)
		return
	}

	err := a.adminRepo.Remove(ctx, u.ID)
	if err != nil {
		a.send(chatID, "❌ Admin ўчиришда хатолик юз берди.")
		return
	}

	a.send(chatID, fmt.Sprintf(MsgAdminRemoved, u.DisplayName))
	a.send(targetTelegramID, MsgAdminRemovedNotify)
}
func (a *AdminHandlers) sendLeagueManageList(ctx context.Context, chatID int64, msgID int) {
	leagues, _ := a.leagueRepo.GetAllLeagues(ctx)
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, l := range leagues {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ "+l.Name, fmt.Sprintf("league_manage_%d", l.ID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(BtnBack, "menu"),
	))
	a.smartUpdate(chatID, msgID, MsgChooseLeagueManage, tgbotapi.NewInlineKeyboardMarkup(rows...))
}
