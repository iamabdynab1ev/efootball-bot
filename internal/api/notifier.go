package api

import (
	"context"
	"efootball-bot/internal/groupcast"
	"efootball-bot/internal/logger"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramNotifier sends Telegram messages on behalf of the API server.
type TelegramNotifier struct {
	bot    *tgbotapi.BotAPI
	groups *groupcast.Hub // групповые каналы (TG-группа, WhatsApp) — может быть nil
}

func NewTelegramNotifier(bot *tgbotapi.BotAPI) *TelegramNotifier {
	return &TelegramNotifier{bot: bot}
}

// SetGroups подключает шину групповых уведомлений: ключевые события
// дублируются в общую группу игроков (Telegram/WhatsApp).
func (n *TelegramNotifier) SetGroups(h *groupcast.Hub) { n.groups = h }

func (n *TelegramNotifier) send(telegramID int64, text string) {
	if n == nil || n.bot == nil || telegramID == 0 {
		return
	}
	msg := tgbotapi.NewMessage(telegramID, text)
	msg.ParseMode = "HTML"
	if _, err := n.bot.Send(msg); err != nil {
		logger.FromContext(context.Background()).Warn("telegram notify failed",
			"telegram_id", telegramID, "error", err)
	}
}

func (n *TelegramNotifier) broadcast(text string, ids []int64) {
	for _, id := range ids {
		n.send(id, text)
	}
}

// BroadcastCustom рассылает произвольное сообщение администратора всем
// привязанным Telegram-аккаунтам и в группу игроков.
func (n *TelegramNotifier) BroadcastCustom(text string, telegramIDs []int64) {
	n.broadcast("📢 <b>Объявление</b>\n\n"+text, telegramIDs)
	n.groups.Publish("📢 Объявление\n\n" + text)
}

// groupScoreText — единый аккуратный формат итога матча для групп
// (одинаковый для обычного подтверждения и админского счёта).
func groupScoreText(homeName, awayName string, homeGoals, awayGoals int16) string {
	return fmt.Sprintf("⚽ Итог матча\n%s  %d : %d  %s", homeName, homeGoals, awayGoals, awayName)
}

// DrawGenerated notifies all league members that the draw was generated.
func (n *TelegramNotifier) DrawGenerated(leagueName string, telegramIDs []int64) {
	n.broadcast(fmt.Sprintf("🎲 <b>%s</b> лигасида жадвал тузилди! Ўйинларингизни текширинг.", leagueName), telegramIDs)
	n.groups.Publish(fmt.Sprintf("🎲 «%s»: жеребьёвка проведена! Расписание матчей уже на сайте.", leagueName))
}

// GroupStageComplete notifies all league members that the group stage has
// finished and the admin can now generate the playoff bracket.
func (n *TelegramNotifier) GroupStageComplete(leagueName string, telegramIDs []int64) {
	n.broadcast(fmt.Sprintf("🏁 <b>%s</b>: групповой этап завершён! Администратор может сгенерировать плей-офф.", leagueName), telegramIDs)
	n.groups.Publish(fmt.Sprintf("🏁 «%s»: групповой этап завершён! Впереди плей-офф.", leagueName))
}

// MemberApproved notifies a player that their league application was approved.
func (n *TelegramNotifier) MemberApproved(leagueName string, telegramID int64) {
	n.send(telegramID, fmt.Sprintf("✅ Сизнинг <b>%s</b> лигасига аризангиз тасдиқланди!", leagueName))
}

// ResultSubmitted notifies the away player that a result was submitted.
func (n *TelegramNotifier) ResultSubmitted(homeName, awayName string, homeGoals, awayGoals int16, awayTelegramID int64) {
	n.send(awayTelegramID, fmt.Sprintf(
		"⚽ <b>%s</b> ўйин натижасини киритди: %d:%d\nТасдиқланг ёки оспорьте қилинг.",
		homeName, homeGoals, awayGoals,
	))
}

// MatchConfirmed notifies both players that the match result was confirmed.
func (n *TelegramNotifier) MatchConfirmed(homeName, awayName string, homeGoals, awayGoals int16, homeTelegramID, awayTelegramID int64) {
	text := fmt.Sprintf("✅ Матч натижаси тасдиқланди: <b>%s %d:%d %s</b>", homeName, homeGoals, awayGoals, awayName)
	n.send(homeTelegramID, text)
	n.send(awayTelegramID, text)
	n.groups.Publish(groupScoreText(homeName, awayName, homeGoals, awayGoals))
}

// MatchDisputed notifies the home player that the result was disputed.
func (n *TelegramNotifier) MatchDisputed(homeName, awayName string, homeGoals, awayGoals int16, homeTelegramID int64) {
	n.send(homeTelegramID, fmt.Sprintf(
		"❌ <b>%s</b> натижани оспорьте қилди (%d:%d). Янги натижа киритинг.",
		awayName, homeGoals, awayGoals,
	))
}

// SendReminder sends a deadline reminder to a player, satisfying service.ReminderNotifier.
func (n *TelegramNotifier) SendReminder(ctx context.Context, telegramID int64, msg string) error {
	if n == nil || n.bot == nil || telegramID == 0 {
		return nil
	}
	tgMsg := tgbotapi.NewMessage(telegramID, msg)
	tgMsg.ParseMode = "HTML"
	_, err := n.bot.Send(tgMsg)
	return err
}

// AdminResolved notifies both players that an admin resolved their match.
func (n *TelegramNotifier) AdminResolved(homeName, awayName string, homeGoals, awayGoals int16, homeTelegramID, awayTelegramID int64) {
	text := fmt.Sprintf("🔧 Администратор матч натижасини белгилади: <b>%s %d:%d %s</b>", homeName, homeGoals, awayGoals, awayName)
	n.send(homeTelegramID, text)
	n.send(awayTelegramID, text)
	n.groups.Publish(groupScoreText(homeName, awayName, homeGoals, awayGoals))
}
