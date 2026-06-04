package handlers

import (
	"html"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func safe(t string) string {
	return html.EscapeString(t)
}
func (h *Handler) smartUpdate(chatID int64, msgID int, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	tgID := chatID

	// 1. Агар msgID = 0 бўлса (пастки тугма босилса), эски хабарни ўчириб янги юборамиз
	if msgID == 0 {
		if oldID, ok := h.menuMsgID.Load(tgID); ok {
			h.bot.Request(tgbotapi.NewDeleteMessage(chatID, oldID.(int)))
		}

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		sent, err := h.bot.Send(msg)
		if err == nil {
			h.menuMsgID.Store(tgID, sent.MessageID) // Янги хабар ИДсини сақлаймиз
		}
		return
	}

	// 2. Агар msgID > 0 бўлса (инлайн тугма босилса), хабарни ўзини таҳрирлаймиз
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, msgID, text, keyboard)
	edit.ParseMode = "HTML"
	_, err := h.bot.Request(edit)

	// Агар таҳрирлашда хато бўлса (масалан хабар ўчиб кетган бўлса), янгисини юборамиз
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = keyboard
		sent, _ := h.bot.Send(msg)
		h.menuMsgID.Store(tgID, sent.MessageID)
	}
}
func (h *Handler) getLastID(tgID int64) int {
	if val, ok := h.menuMsgID.Load(tgID); ok {
		return val.(int)
	}
	return 0
}
func (h *Handler) isMenuButton(t string) bool {
	return t == MenuMain || t == MenuStandings || t == MenuSchedule || t == MenuProfile ||
		t == MenuAdminCreate || t == MenuAdminLeagues || t == MenuAdminPending || t == MenuAdminDraw ||
		t == MenuJoinLeague || t == MenuResult || t == MenuHistory || t == MenuAdminDispute ||
		t == MenuAdminManage || t == MenuHelp || t == MenuCard
}
func (h *Handler) sendChattable(ch tgbotapi.Chattable) (tgbotapi.Message, error) {
	m, e := h.bot.Send(ch)
	if e != nil {
		log.Printf("telegram send error: type=%T err=%v", ch, e)
	}
	return m, e
}
func (h *Handler) send(chatID int64, text string) {
	m := tgbotapi.NewMessage(chatID, text)
	m.ParseMode = "HTML"
	h.sendChattable(m)
}
