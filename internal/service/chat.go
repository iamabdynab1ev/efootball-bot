package service

import (
	"context"
	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"errors"
	"strings"
)

// ChatService — чат турнира. Доставка адресная: сообщение публикуется в личные
// SSE-топики участников комнаты (членство выводится из league_members), поэтому
// динамические подписки на комнаты не нужны. Целостность (no-loss) обеспечивает
// персистентность + догрузка по id на фронте, живой канал — best-effort сверху.
type ChatService struct {
	chatRepo   repository.ChatRepository
	leagueRepo repository.LeagueRepository
	// publish доставляет событие в личные топики перечисленных пользователей.
	publish func(userIDs []int64, eventType string, data any)
}

const chatMaxBody = 2000

var (
	ErrChatForbidden = errors.New("нет доступа к этому чату")
	ErrChatEmpty     = errors.New("пустое сообщение")
	ErrChatArchived  = errors.New("чат архивирован")
)

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
	return msg, nil
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
