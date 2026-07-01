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
	// EnsureDirectRoom находит-или-создаёт ЛС-комнату двух пользователей
	// (нормализованная пара). Идемпотентно.
	EnsureDirectRoom(ctx context.Context, userA, userB int64) (*models.ChatRoom, error)
	// ListDirectRooms — диалоги пользователя с собеседником и последним сообщением.
	ListDirectRooms(ctx context.Context, userID int64) ([]*models.DirectRoomView, error)
	// AreOpponents — были/есть ли эти двое соперниками хотя бы в одном матче.
	AreOpponents(ctx context.Context, userA, userB int64) (bool, error)
	// MarkRead поднимает отметку прочтения комнаты пользователем до uptoID и
	// возвращает актуальный last_read (монотонно, не откатывается назад).
	MarkRead(ctx context.Context, userID, roomID, uptoID int64) (int64, error)
	// UnreadTotalDirect — всего непрочитанных ЛС у пользователя (для бейджа).
	UnreadTotalDirect(ctx context.Context, userID int64) (int, error)
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

// roomCols — единый список колонок комнаты (league_id может быть NULL у ЛС).
const roomCols = `id, COALESCE(league_id, 0), group_name, title, archived, created_at, kind, dm_lo, dm_hi`

func scanRoom(row interface {
	Scan(dest ...any) error
}) (*models.ChatRoom, error) {
	r := &models.ChatRoom{}
	if err := row.Scan(&r.ID, &r.LeagueID, &r.GroupName, &r.Title, &r.Archived, &r.CreatedAt,
		&r.Kind, &r.DmLo, &r.DmHi); err != nil {
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
		RETURNING id, COALESCE(league_id, 0), group_name, title, archived, created_at, kind, dm_lo, dm_hi
	`, leagueID, groupName, title)
	return scanRoom(row)
}

func (r *chatRepo) GetRoom(ctx context.Context, roomID int64) (*models.ChatRoom, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+roomCols+`
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
		SELECT `+roomCols+`
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
		SELECT r.id, COALESCE(r.league_id, 0), r.group_name, r.title, r.archived, r.created_at, r.kind, r.dm_lo, r.dm_hi
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
			WHERE r.id = $1 AND (
				-- ЛС: доступ ровно у двух участников пары.
				(r.kind = 'direct' AND $2 IN (r.dm_lo, r.dm_hi))
				-- Групповой чат: членство из league_members (approved + группа).
				OR (r.kind = 'group' AND EXISTS (
					SELECT 1 FROM league_members lm
					WHERE lm.league_id = r.league_id AND lm.user_id = $2 AND lm.status = 'approved'
					  AND (r.group_name = '' OR r.group_name = COALESCE(lm.group_name, ''))
				))
			)
		)
	`, roomID, userID).Scan(&ok)
	return ok, err
}

func (r *chatRepo) RoomMemberIDs(ctx context.Context, roomID int64) ([]int64, error) {
	rows, err := r.db.Query(ctx, `
		-- ЛС: двое из пары. Групповой: члены лиги/группы.
		SELECT uid FROM (
			SELECT r.dm_lo AS uid FROM chat_rooms r WHERE r.id = $1 AND r.kind = 'direct' AND r.dm_lo IS NOT NULL
			UNION
			SELECT r.dm_hi FROM chat_rooms r WHERE r.id = $1 AND r.kind = 'direct' AND r.dm_hi IS NOT NULL
			UNION
			SELECT lm.user_id FROM chat_rooms r
			JOIN league_members lm
			  ON lm.league_id = r.league_id AND lm.status = 'approved'
			WHERE r.id = $1 AND r.kind = 'group'
			  AND (r.group_name = '' OR r.group_name = COALESCE(lm.group_name, ''))
		) m
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

func (r *chatRepo) EnsureDirectRoom(ctx context.Context, userA, userB int64) (*models.ChatRoom, error) {
	lo, hi := userA, userB
	if lo > hi {
		lo, hi = hi, lo
	}
	// DO UPDATE (no-op) — чтобы RETURNING сработал и при уже существующей паре.
	row := r.db.QueryRow(ctx, `
		INSERT INTO chat_rooms (league_id, group_name, title, kind, dm_lo, dm_hi)
		VALUES (NULL, '', '', 'direct', $1, $2)
		ON CONFLICT (dm_lo, dm_hi) WHERE kind = 'direct'
		DO UPDATE SET title = chat_rooms.title
		RETURNING id, COALESCE(league_id, 0), group_name, title, archived, created_at, kind, dm_lo, dm_hi
	`, lo, hi)
	return scanRoom(row)
}

func (r *chatRepo) ListDirectRooms(ctx context.Context, userID int64) ([]*models.DirectRoomView, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id,
		       other.id, other.display_name, COALESCE(other.favorite_club, ''), other.last_seen_at,
		       last.body, last.deleted, last.created_at, last.user_id,
		       -- непрочитанные мной: чужие сообщения новее моей отметки прочтения
		       (SELECT count(*) FROM chat_messages m
		          WHERE m.room_id = r.id AND m.user_id <> $1 AND NOT m.deleted
		            AND m.id > COALESCE(mine.last_read_id, 0)) AS unread,
		       COALESCE(theirs.last_read_id, 0) AS other_last_read
		FROM chat_rooms r
		JOIN users other
		  ON other.id = CASE WHEN r.dm_lo = $1 THEN r.dm_hi ELSE r.dm_lo END
		LEFT JOIN chat_reads mine   ON mine.room_id = r.id AND mine.user_id = $1
		LEFT JOIN chat_reads theirs ON theirs.room_id = r.id AND theirs.user_id = other.id
		LEFT JOIN LATERAL (
			SELECT m.body, m.deleted, m.created_at, m.user_id
			FROM chat_messages m WHERE m.room_id = r.id
			ORDER BY m.id DESC LIMIT 1
		) last ON TRUE
		WHERE r.kind = 'direct' AND $1 IN (r.dm_lo, r.dm_hi)
		ORDER BY COALESCE(last.created_at, r.created_at) DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.DirectRoomView
	for rows.Next() {
		v := &models.DirectRoomView{}
		var body *string
		var deleted *bool
		if err := rows.Scan(&v.RoomID, &v.OtherID, &v.OtherName, &v.OtherClub, &v.OtherLastSeen,
			&body, &deleted, &v.LastAt, &v.LastAuthorID, &v.Unread, &v.OtherLastRead); err != nil {
			return nil, err
		}
		if body != nil && !(deleted != nil && *deleted) {
			v.LastBody = *body
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *chatRepo) MarkRead(ctx context.Context, userID, roomID, uptoID int64) (int64, error) {
	var lastRead int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO chat_reads (room_id, user_id, last_read_id, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (room_id, user_id)
		DO UPDATE SET last_read_id = GREATEST(chat_reads.last_read_id, EXCLUDED.last_read_id),
		              updated_at = NOW()
		RETURNING last_read_id
	`, roomID, userID, uptoID).Scan(&lastRead)
	return lastRead, err
}

func (r *chatRepo) UnreadTotalDirect(ctx context.Context, userID int64) (int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(count(*), 0)
		FROM chat_rooms r
		JOIN chat_messages m ON m.room_id = r.id AND m.user_id <> $1 AND NOT m.deleted
		LEFT JOIN chat_reads cr ON cr.room_id = r.id AND cr.user_id = $1
		WHERE r.kind = 'direct' AND $1 IN (r.dm_lo, r.dm_hi)
		  AND m.id > COALESCE(cr.last_read_id, 0)
	`, userID).Scan(&total)
	return total, err
}

func (r *chatRepo) AreOpponents(ctx context.Context, userA, userB int64) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM matches
			WHERE (home_user_id = $1 AND away_user_id = $2)
			   OR (home_user_id = $2 AND away_user_id = $1)
		)
	`, userA, userB).Scan(&ok)
	return ok, err
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
