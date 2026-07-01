package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"errors"
	"regexp"
	"strings"
)

// mentionRe — @упоминание: буквы (в т.ч. кириллица), цифры, подчёркивание.
var mentionRe = regexp.MustCompile(`@([\p{L}\p{N}_]+)`)

// ChatService — чат турнира. Доставка адресная: сообщение публикуется в личные
// SSE-топики участников комнаты (членство выводится из league_members), поэтому
// динамические подписки на комнаты не нужны. Целостность (no-loss) обеспечивает
// персистентность + догрузка по id на фронте, живой канал — best-effort сверху.
type ChatService struct {
	chatRepo   repository.ChatRepository
	leagueRepo repository.LeagueRepository
	// publish доставляет событие в личные топики перечисленных пользователей.
	publish func(userIDs []int64, eventType string, data any)
	// onMention вызывается при @упоминании участников (колокольчик + пуш). nil —
	// упоминания только подсвечиваются, без уведомлений.
	onMention func(ctx context.Context, msg *models.ChatMessage, mentionedIDs []int64, leagueID int64)
	// onDirect вызывается при новом личном сообщении — уведомляет собеседника.
	onDirect func(ctx context.Context, msg *models.ChatMessage, recipientID int64)
}

// SetMentionHandler подключает обработчик @упоминаний (уведомления/пуш).
func (s *ChatService) SetMentionHandler(fn func(ctx context.Context, msg *models.ChatMessage, mentionedIDs []int64, leagueID int64)) {
	s.onMention = fn
}

// SetDirectHandler подключает обработчик личных сообщений (уведомление собеседнику).
func (s *ChatService) SetDirectHandler(fn func(ctx context.Context, msg *models.ChatMessage, recipientID int64)) {
	s.onDirect = fn
}

const chatMaxBody = 2000

var (
	ErrChatForbidden    = errors.New("нет доступа к этому чату")
	ErrChatEmpty        = errors.New("пустое сообщение")
	ErrChatArchived     = errors.New("чат архивирован")
	ErrChatNotOpponents = errors.New("писать можно только соперникам по матчу")
)

// OpenDirect находит-или-создаёт личную комнату с соперником. Писать можно
// только тем, с кем есть общий матч (защита от спама произвольным юзерам).
func (s *ChatService) OpenDirect(ctx context.Context, requester, target int64) (*models.ChatRoom, error) {
	if requester == target || target <= 0 {
		return nil, ErrChatForbidden
	}
	ok, err := s.chatRepo.AreOpponents(ctx, requester, target)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrChatNotOpponents
	}
	return s.chatRepo.EnsureDirectRoom(ctx, requester, target)
}

// ListDirect — диалоги пользователя (для экрана «Сообщения»).
func (s *ChatService) ListDirect(ctx context.Context, userID int64) ([]*models.DirectRoomView, error) {
	return s.chatRepo.ListDirectRooms(ctx, userID)
}

// MarkRead двигает отметку прочтения комнаты и оповещает собеседника (для ✓✓).
func (s *ChatService) MarkRead(ctx context.Context, userID, roomID, uptoID int64) (int64, error) {
	ok, err := s.chatRepo.CanAccessRoom(ctx, userID, roomID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrChatForbidden
	}
	lastRead, err := s.chatRepo.MarkRead(ctx, userID, roomID, uptoID)
	if err != nil {
		return 0, err
	}
	// Оповещаем других участников (автор увидит ✓✓ на своих сообщениях).
	s.fanoutExcept(ctx, roomID, userID, "chat_read", map[string]any{
		"room_id": roomID, "user_id": userID, "last_read_id": lastRead,
	})
	return lastRead, nil
}

// Typing рассылает эфемерный сигнал «печатает…» другим участникам комнаты.
func (s *ChatService) Typing(ctx context.Context, userID, roomID int64) error {
	ok, err := s.chatRepo.CanAccessRoom(ctx, userID, roomID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrChatForbidden
	}
	s.fanoutExcept(ctx, roomID, userID, "chat_typing", map[string]any{
		"room_id": roomID, "user_id": userID,
	})
	return nil
}

// UnreadTotal — всего непрочитанных ЛС (для бейджа на иконке «Сообщения»).
func (s *ChatService) UnreadTotal(ctx context.Context, userID int64) (int, error) {
	return s.chatRepo.UnreadTotalDirect(ctx, userID)
}

func NewChatService(chatRepo repository.ChatRepository, leagueRepo repository.LeagueRepository, publish func(userIDs []int64, eventType string, data any)) *ChatService {
	return &ChatService{chatRepo: chatRepo, leagueRepo: leagueRepo, publish: publish}
}

// EnsureRoomsForLeague идемпотентно создаёт общую комнату лиги и по комнате на
// каждую группу. Вызывается при жеребьёвке и лениво при открытии чата.
func (s *ChatService) EnsureRoomsForLeague(ctx context.Context, leagueID int64) error {
	if _, err := s.chatRepo.EnsureRoom(ctx, leagueID, "", "Общий чат"); err != nil {
		return err
	}
	groups, err := s.leagueRepo.GetLeagueGroups(ctx, leagueID)
	if err != nil {
		return err
	}
	for _, g := range groups {
		if g == "" {
			continue
		}
		if _, err := s.chatRepo.EnsureRoom(ctx, leagueID, g, "Группа "+g); err != nil {
			return err
		}
	}
	return nil
}

// RoomsForUser гарантирует наличие комнат и возвращает доступные пользователю.
func (s *ChatService) RoomsForUser(ctx context.Context, userID, leagueID int64) ([]*models.ChatRoom, error) {
	if err := s.EnsureRoomsForLeague(ctx, leagueID); err != nil {
		return nil, err
	}
	return s.chatRepo.ListAccessibleRooms(ctx, userID, leagueID)
}

// History возвращает сообщения комнаты (проверив доступ). since>0 — догрузка
// новых при реконнекте (no-loss), before>0 — листание истории вверх.
func (s *ChatService) History(ctx context.Context, userID, roomID, beforeID, sinceID int64, limit int) ([]*models.ChatMessage, error) {
	ok, err := s.chatRepo.CanAccessRoom(ctx, userID, roomID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrChatForbidden
	}
	return s.chatRepo.ListMessages(ctx, roomID, beforeID, sinceID, limit)
}

// Members возвращает участников комнаты (с проверкой доступа) — для
// @упоминаний: скоуп строго по комнате (общая = вся лига, группа = её игроки).
func (s *ChatService) Members(ctx context.Context, userID, roomID int64) ([]*models.ChatMember, error) {
	ok, err := s.chatRepo.CanAccessRoom(ctx, userID, roomID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrChatForbidden
	}
	return s.chatRepo.RoomMembers(ctx, roomID)
}

// Send проверяет доступ и архив, сохраняет сообщение и доставляет его всем
// участникам комнаты в реальном времени.
func (s *ChatService) Send(ctx context.Context, userID, roomID int64, body string) (*models.ChatMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrChatEmpty
	}
	if len(body) > chatMaxBody {
		body = body[:chatMaxBody]
	}

	room, err := s.chatRepo.GetRoom(ctx, roomID)
	if err != nil || room == nil {
		return nil, ErrChatForbidden
	}
	if room.Archived {
		return nil, ErrChatArchived
	}
	ok, err := s.chatRepo.CanAccessRoom(ctx, userID, roomID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrChatForbidden
	}

	msg, err := s.chatRepo.InsertMessage(ctx, roomID, userID, body)
	if err != nil {
		return nil, err
	}
	s.fanout(ctx, roomID, "chat", msg)

	if room.Kind == "direct" {
		// ЛС: уведомляем собеседника (второго из пары).
		if s.onDirect != nil && room.DmLo != nil && room.DmHi != nil {
			recipient := *room.DmLo
			if recipient == userID {
				recipient = *room.DmHi
			}
			s.onDirect(ctx, msg, recipient)
		}
	} else if s.onMention != nil && strings.Contains(body, "@") {
		// @упоминания: уведомляем упомянутых участников комнаты (кроме автора).
		if ids := s.resolveMentions(ctx, roomID, body, userID); len(ids) > 0 {
			s.onMention(ctx, msg, ids, room.LeagueID)
		}
	}
	return msg, nil
}

// resolveMentions сопоставляет @токены с участниками комнаты по ИМЕНИ (первое
// слово display_name, регистронезависимо — фронт вставляет @Имя). Возвращает id
// упомянутых участников, кроме автора, без дублей.
func (s *ChatService) resolveMentions(ctx context.Context, roomID int64, body string, authorID int64) []int64 {
	matches := mentionRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	tokens := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		tokens[strings.ToLower(m[1])] = struct{}{}
	}
	members, err := s.chatRepo.RoomMembers(ctx, roomID)
	if err != nil {
		return nil
	}
	var ids []int64
	seen := map[int64]struct{}{}
	for _, mem := range members {
		if mem.UserID == authorID {
			continue
		}
		first := mem.DisplayName
		if i := strings.IndexByte(first, ' '); i > 0 {
			first = first[:i]
		}
		if _, ok := tokens[strings.ToLower(first)]; !ok {
			continue
		}
		if _, dup := seen[mem.UserID]; dup {
			continue
		}
		seen[mem.UserID] = struct{}{}
		ids = append(ids, mem.UserID)
	}
	return ids
}

// DeleteMessage (админ) помечает сообщение удалённым и рассылает дельту.
func (s *ChatService) DeleteMessage(ctx context.Context, messageID int64) (*models.ChatMessage, error) {
	msg, err := s.chatRepo.DeleteMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}
	s.fanout(ctx, msg.RoomID, "chat_deleted", map[string]any{"room_id": msg.RoomID, "id": msg.ID})
	return msg, nil
}

// ListRooms (админ) — все комнаты лиги.
func (s *ChatService) ListRooms(ctx context.Context, leagueID int64) ([]*models.ChatRoom, error) {
	return s.chatRepo.ListRoomsForLeague(ctx, leagueID)
}

// AdminMessages (админ) — сообщения комнаты без проверки членства.
func (s *ChatService) AdminMessages(ctx context.Context, roomID, beforeID, sinceID int64, limit int) ([]*models.ChatMessage, error) {
	return s.chatRepo.ListMessages(ctx, roomID, beforeID, sinceID, limit)
}

// Archive архивирует все комнаты лиги (по завершении турнира — сохраняем, а не
// удаляем переписку).
func (s *ChatService) Archive(ctx context.Context, leagueID int64) error {
	return s.chatRepo.ArchiveRoomsForLeague(ctx, leagueID)
}

// fanout доставляет событие участникам комнаты в их личные топики.
func (s *ChatService) fanout(ctx context.Context, roomID int64, eventType string, data any) {
	if s.publish == nil {
		return
	}
	ids, err := s.chatRepo.RoomMemberIDs(ctx, roomID)
	if err != nil || len(ids) == 0 {
		return
	}
	s.publish(ids, eventType, data)
}

// fanoutExcept — как fanout, но без указанного пользователя (например, автор не
// должен получать собственный сигнал «прочитано»/«печатает»).
func (s *ChatService) fanoutExcept(ctx context.Context, roomID, exceptUserID int64, eventType string, data any) {
	if s.publish == nil {
		return
	}
	ids, err := s.chatRepo.RoomMemberIDs(ctx, roomID)
	if err != nil {
		return
	}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != exceptUserID {
			out = append(out, id)
		}
	}
	if len(out) > 0 {
		s.publish(out, eventType, data)
	}
}
