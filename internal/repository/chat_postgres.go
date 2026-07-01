package repository

import (
	"context"
	"efootball-bot/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository interface {
	// EnsureRoom создаёт комнату (идемпотентно по league_id+group_name) и
	// возвращает её — для авто-создания комнат группы/лиги.
	EnsureRoom(ctx context.Context, leagueID int64, groupName, title string) (*models.ChatRoom, error)
	GetRoom(ctx context.Context, roomID int64) (*models.ChatRoom, error)
	ListRoomsForLeague(ctx context.Context, leagueID int64) ([]*models.ChatRoom, error)
	// ListAccessibleRooms — только комнаты лиги, доступные пользователю (общая +
	// его группа), одним запросом.
	ListAccessibleRooms(ctx context.Context, userID, leagueID int64) ([]*models.ChatRoom, error)
	// CanAccessRoom — членство выводится из league_members (approved + совпадение
	// группы), отдельной таблицы участников нет.
	CanAccessRoom(ctx context.Context, userID, roomID int64) (bool, error)
	// RoomMemberIDs — id участников комнаты (для адресной живой доставки в их
	// личные SSE-топики).
	RoomMemberIDs(ctx context.Context, roomID int64) ([]int64, error)
	// RoomMembers — участники комнаты с именами (для @упоминаний, скоуп по комнате).
	RoomMembers(ctx context.Context, roomID int64) ([]*models.ChatMember, error)
	InsertMessage(ctx context.Context, roomID, userID int64, body string) (*models.ChatMessage, error)
	// ListMessages: since>0 — сообщения новее since (catch-up по возрастанию id);
	// иначе последние (или старше before) — для истории. Всегда по возрастанию id.
	ListMessages(ctx context.Context, roomID, beforeID, sinceID int64, limit int) ([]*models.ChatMessage, error)
	DeleteMessage(ctx context.Context, messageID int64) (*models.ChatMessage, error)
	ArchiveRoomsForLeague(ctx context.Context, leagueID int64) error
}

type chatRepo struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) ChatRepository {
	return &chatRepo{db: db}
}

func scanRoom(row interface {
	Scan(dest ...any) error
}) (*models.ChatRoom, error) {
	r := &models.ChatRoom{}
	if err := row.Scan(&r.ID, &r.LeagueID, &r.GroupName, &r.Title, &r.Archived, &r.CreatedAt); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *chatRepo) EnsureRoom(ctx context.Context, leagueID int64, groupName, title string) (*models.ChatRoom, error) {
	// DO UPDATE (no-op) чтобы RETURNING сработал и при конфликте.
	row := r.db.QueryRow(ctx, `
		INSERT INTO chat_rooms (league_id, group_name, title)
		VALUES ($1, $2, $3)
		ON CONFLICT (league_id, group_name) DO UPDATE SET title = chat_rooms.title
		RETURNING id, league_id, group_name, title, archived, created_at
	`, leagueID, groupName, title)
	return scanRoom(row)
}

func (r *chatRepo) GetRoom(ctx context.Context, roomID int64) (*models.ChatRoom, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, league_id, group_name, title, archived, created_at
		FROM chat_rooms WHERE id = $1
	`, roomID)
	room, err := scanRoom(row)
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (r *chatRepo) ListRoomsForLeague(ctx context.Context, leagueID int64) ([]*models.ChatRoom, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, league_id, group_name, title, archived, created_at
		FROM chat_rooms WHERE league_id = $1
		ORDER BY group_name
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.ChatRoom
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, room)
	}
	return out, rows.Err()
}

func (r *chatRepo) ListAccessibleRooms(ctx context.Context, userID, leagueID int64) ([]*models.ChatRoom, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id, r.league_id, r.group_name, r.title, r.archived, r.created_at
		FROM chat_rooms r
		JOIN league_members lm
		  ON lm.league_id = r.league_id AND lm.user_id = $2 AND lm.status = 'approved'
		WHERE r.league_id = $1
		  AND (r.group_name = '' OR r.group_name = COALESCE(lm.group_name, ''))
		ORDER BY r.group_name
	`, leagueID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.ChatRoom
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, room)
	}
	return out, rows.Err()
}

func (r *chatRepo) CanAccessRoom(ctx context.Context, userID, roomID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM chat_rooms r
			JOIN league_members lm
			  ON lm.league_id = r.league_id AND lm.user_id = $2 AND lm.status = 'approved'
			WHERE r.id = $1
			  AND (r.group_name = '' OR r.group_name = COALESCE(lm.group_name, ''))
		)
	`, roomID, userID).Scan(&ok)
	return ok, err
}

func (r *chatRepo) RoomMemberIDs(ctx context.Context, roomID int64) ([]int64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.user_id FROM chat_rooms r
		JOIN league_members lm
		  ON lm.league_id = r.league_id AND lm.status = 'approved'
		WHERE r.id = $1
		  AND (r.group_name = '' OR r.group_name = COALESCE(lm.group_name, ''))
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *chatRepo) RoomMembers(ctx context.Context, roomID int64) ([]*models.ChatMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.user_id, u.display_name, COALESCE(u.favorite_club, '')
		FROM chat_rooms r
		JOIN league_members lm
		  ON lm.league_id = r.league_id AND lm.status = 'approved'
		JOIN users u ON u.id = lm.user_id
		WHERE r.id = $1
		  AND (r.group_name = '' OR r.group_name = COALESCE(lm.group_name, ''))
		ORDER BY u.display_name
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.ChatMember
	for rows.Next() {
		m := &models.ChatMember{}
		if err := rows.Scan(&m.UserID, &m.DisplayName, &m.FavoriteClub); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *chatRepo) InsertMessage(ctx context.Context, roomID, userID int64, body string) (*models.ChatMessage, error) {
	// INSERT + JOIN автора одним round-trip'ом.
	row := r.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO chat_messages (room_id, user_id, body)
			VALUES ($1, $2, $3)
			RETURNING id, room_id, user_id, body, deleted, created_at
		)
		SELECT ins.id, ins.room_id, ins.user_id,
		       COALESCE(u.display_name, ''), COALESCE(u.favorite_club, ''),
		       ins.body, ins.deleted, ins.created_at
		FROM ins LEFT JOIN users u ON u.id = ins.user_id
	`, roomID, userID, body)
	return scanMessage(row)
}

func (r *chatRepo) ListMessages(ctx context.Context, roomID, beforeID, sinceID int64, limit int) ([]*models.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var (
		query string
		args  []any
		desc  bool // выбрали по убыванию (history) — перевернём в конце
	)
	switch {
	case sinceID > 0:
		// Catch-up: новее since, по возрастанию.
		query = `SELECT m.id, m.room_id, m.user_id, COALESCE(u.display_name,''),
			        COALESCE(u.favorite_club,''), m.body, m.deleted, m.created_at
			FROM chat_messages m LEFT JOIN users u ON u.id = m.user_id
			WHERE m.room_id = $1 AND m.id > $2 ORDER BY m.id ASC LIMIT $3`
		args = []any{roomID, sinceID, limit}
	case beforeID > 0:
		query = `SELECT m.id, m.room_id, m.user_id, COALESCE(u.display_name,''),
			        COALESCE(u.favorite_club,''), m.body, m.deleted, m.created_at
			FROM chat_messages m LEFT JOIN users u ON u.id = m.user_id
			WHERE m.room_id = $1 AND m.id < $2 ORDER BY m.id DESC LIMIT $3`
		args = []any{roomID, beforeID, limit}
		desc = true
	default:
		query = `SELECT m.id, m.room_id, m.user_id, COALESCE(u.display_name,''),
			        COALESCE(u.favorite_club,''), m.body, m.deleted, m.created_at
			FROM chat_messages m LEFT JOIN users u ON u.id = m.user_id
			WHERE m.room_id = $1 ORDER BY m.id DESC LIMIT $2`
		args = []any{roomID, limit}
		desc = true
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.ChatMessage
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if desc {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out, nil
}

func (r *chatRepo) DeleteMessage(ctx context.Context, messageID int64) (*models.ChatMessage, error) {
	row := r.db.QueryRow(ctx, `
		WITH upd AS (
			UPDATE chat_messages SET deleted = TRUE WHERE id = $1
			RETURNING id, room_id, user_id, body, deleted, created_at
		)
		SELECT upd.id, upd.room_id, upd.user_id,
		       COALESCE(u.display_name,''), COALESCE(u.favorite_club,''),
		       upd.body, upd.deleted, upd.created_at
		FROM upd LEFT JOIN users u ON u.id = upd.user_id
	`, messageID)
	return scanMessage(row)
}

func (r *chatRepo) ArchiveRoomsForLeague(ctx context.Context, leagueID int64) error {
	_, err := r.db.Exec(ctx, `UPDATE chat_rooms SET archived = TRUE WHERE league_id = $1`, leagueID)
	return err
}

func scanMessage(row interface {
	Scan(dest ...any) error
}) (*models.ChatMessage, error) {
	m := &models.ChatMessage{}
	if err := row.Scan(&m.ID, &m.RoomID, &m.UserID, &m.AuthorName, &m.AuthorClub,
		&m.Body, &m.Deleted, &m.CreatedAt); err != nil {
		return nil, err
	}
	// Удалённые сообщения не отдаём телом — только пометку.
	if m.Deleted {
		m.Body = ""
	}
	return m, nil
}
