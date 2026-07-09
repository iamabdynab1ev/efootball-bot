package repository

import (
	"context"
	"efootball-bot/internal/models"
	"encoding/json"

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
	// RoomReads — до какого сообщения дочитал каждый участник комнаты (0, если не
	// читал). Для отметок «прочитано» в групповом чате.
	RoomReads(ctx context.Context, roomID int64) ([]models.RoomRead, error)
	// RoomMembers — участники комнаты с именами (для @упоминаний, скоуп по комнате).
	RoomMembers(ctx context.Context, roomID int64) ([]*models.ChatMember, error)
	// EnsureDirectRoom находит-или-создаёт ЛС-комнату двух пользователей
	// (нормализованная пара). Идемпотентно.
	EnsureDirectRoom(ctx context.Context, userA, userB int64) (*models.ChatRoom, error)
	// ListDirectRooms — диалоги пользователя с собеседником и последним сообщением.
	ListDirectRooms(ctx context.Context, userID int64) ([]*models.DirectRoomView, error)
	// ClearDirectForMe — «удалить чат у меня»: скрыть текущую историю диалога
	// для одного участника (собеседник ничего не замечает).
	ClearDirectForMe(ctx context.Context, userID, roomID int64) error
	// DeleteDirectRoom — «удалить у обоих»: комната и вся переписка удаляются
	// целиком (CASCADE). Только для kind='direct'.
	DeleteDirectRoom(ctx context.Context, roomID int64) error
	// ClearPoint — точка очистки истории (0, если пользователь чат не удалял).
	ClearPoint(ctx context.Context, userID, roomID int64) (int64, error)
	// AreOpponents — были/есть ли эти двое соперниками хотя бы в одном матче.
	AreOpponents(ctx context.Context, userA, userB int64) (bool, error)
	// MarkRead поднимает отметку прочтения комнаты пользователем до uptoID и
	// возвращает актуальный last_read (монотонно, не откатывается назад).
	MarkRead(ctx context.Context, userID, roomID, uptoID int64) (int64, error)
	// UnreadTotalDirect — всего непрочитанных ЛС у пользователя (для бейджа).
	UnreadTotalDirect(ctx context.Context, userID int64) (int, error)
	InsertMessage(ctx context.Context, roomID, userID int64, body string, replyToID *int64, media *models.ChatMedia) (*models.ChatMessage, error)
	// Реакции на сообщения (эмодзи). Возвращают changed=true, только если строка
	// реально добавлена/удалена (чтобы не рассылать «пустые» события).
	AddReaction(ctx context.Context, messageID, userID int64, emoji string) (prev string, inserted bool, err error)
	RemoveReaction(ctx context.Context, messageID, userID int64, emoji string) (bool, error)
	UserBrief(ctx context.Context, userID int64) (name, club string, err error)
	// RoomReactions — агрегированные реакции всех сообщений комнаты (+ mine для userID).
	RoomReactions(ctx context.Context, roomID, userID int64) ([]models.ReactionAgg, error)
	// ListMessages: since>0 — сообщения новее since (catch-up по возрастанию id);
	// иначе последние (или старше before) — для истории. Всегда по возрастанию id.
	// minID>0 скрывает сообщения с id<=minID (очистка «удалить чат у меня»).
	ListMessages(ctx context.Context, roomID, beforeID, sinceID int64, limit int, minID int64) ([]*models.ChatMessage, error)
	DeleteMessage(ctx context.Context, messageID int64) (*models.ChatMessage, error)
	// MessageMeta — автор и комната сообщения (для проверки прав на правку/удаление).
	MessageMeta(ctx context.Context, messageID int64) (authorID, roomID int64, err error)
	// UpdateMessage меняет текст сообщения и ставит пометку edited.
	UpdateMessage(ctx context.Context, messageID int64, body string) (*models.ChatMessage, error)
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
		SELECT r.id, COALESCE(r.league_id, 0), r.group_name, r.title, r.archived, r.created_at, r.kind, r.dm_lo, r.dm_hi,
		       (SELECT count(*) FROM chat_messages m
		          WHERE m.room_id = r.id AND m.user_id <> $2 AND NOT m.deleted
		            AND m.id > COALESCE((SELECT last_read_id FROM chat_reads cr
		                                 WHERE cr.room_id = r.id AND cr.user_id = $2), 0)) AS unread
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
		room := &models.ChatRoom{}
		if err := rows.Scan(&room.ID, &room.LeagueID, &room.GroupName, &room.Title, &room.Archived,
			&room.CreatedAt, &room.Kind, &room.DmLo, &room.DmHi, &room.Unread); err != nil {
			return nil, err
		}
		out = append(out, room)
	}
	return out, rows.Err()
}

func (r *chatRepo) RoomReads(ctx context.Context, roomID int64) ([]models.RoomRead, error) {
	rows, err := r.db.Query(ctx, `
		SELECT mem.uid, COALESCE(cr.last_read_id, 0)
		FROM (
			SELECT r.dm_lo AS uid FROM chat_rooms r WHERE r.id = $1 AND r.kind = 'direct' AND r.dm_lo IS NOT NULL
			UNION
			SELECT r.dm_hi FROM chat_rooms r WHERE r.id = $1 AND r.kind = 'direct' AND r.dm_hi IS NOT NULL
			UNION
			SELECT lm.user_id FROM chat_rooms r
			JOIN league_members lm
			  ON lm.league_id = r.league_id AND lm.status = 'approved'
			WHERE r.id = $1 AND r.kind = 'group'
			  AND (r.group_name = '' OR r.group_name = COALESCE(lm.group_name, ''))
		) mem
		LEFT JOIN chat_reads cr ON cr.room_id = $1 AND cr.user_id = mem.uid
	`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.RoomRead
	for rows.Next() {
		var rr models.RoomRead
		if err := rows.Scan(&rr.UserID, &rr.LastRead); err != nil {
			return nil, err
		}
		out = append(out, rr)
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
	// cl.upto_id — точка «удалить чат у меня»: всё, что старше, для этого
	// пользователя не существует; диалог без новых сообщений скрыт из списка.
	rows, err := r.db.Query(ctx, `
		SELECT r.id,
		       other.id, other.display_name, COALESCE(other.favorite_club, ''), other.last_seen_at,
		       last.body, last.deleted, last.created_at, last.user_id,
		       -- непрочитанные мной: чужие сообщения новее моей отметки прочтения
		       (SELECT count(*) FROM chat_messages m
		          WHERE m.room_id = r.id AND m.user_id <> $1 AND NOT m.deleted
		            AND m.id > GREATEST(COALESCE(mine.last_read_id, 0), COALESCE(cl.upto_id, 0))) AS unread,
		       COALESCE(theirs.last_read_id, 0) AS other_last_read
		FROM chat_rooms r
		JOIN users other
		  ON other.id = CASE WHEN r.dm_lo = $1 THEN r.dm_hi ELSE r.dm_lo END
		LEFT JOIN chat_reads mine   ON mine.room_id = r.id AND mine.user_id = $1
		LEFT JOIN chat_reads theirs ON theirs.room_id = r.id AND theirs.user_id = other.id
		LEFT JOIN chat_clears cl    ON cl.room_id = r.id AND cl.user_id = $1
		LEFT JOIN LATERAL (
			SELECT m.body, m.deleted, m.created_at, m.user_id
			FROM chat_messages m
			WHERE m.room_id = r.id AND m.id > COALESCE(cl.upto_id, 0)
			ORDER BY m.id DESC LIMIT 1
		) last ON TRUE
		WHERE r.kind = 'direct' AND $1 IN (r.dm_lo, r.dm_hi)
		  AND (cl.upto_id IS NULL OR last.created_at IS NOT NULL)
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
		LEFT JOIN chat_clears cl ON cl.room_id = r.id AND cl.user_id = $1
		WHERE r.kind = 'direct' AND $1 IN (r.dm_lo, r.dm_hi)
		  AND m.id > GREATEST(COALESCE(cr.last_read_id, 0), COALESCE(cl.upto_id, 0))
	`, userID).Scan(&total)
	return total, err
}

// ClearDirectForMe — «удалить чат у меня»: фиксируем точку очистки на текущем
// последнем сообщении. Повторное удаление сдвигает точку вперёд.
func (r *chatRepo) ClearDirectForMe(ctx context.Context, userID, roomID int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO chat_clears (room_id, user_id, upto_id)
		VALUES ($1, $2, COALESCE((SELECT max(id) FROM chat_messages WHERE room_id = $1), 0))
		ON CONFLICT (room_id, user_id)
		DO UPDATE SET upto_id = GREATEST(chat_clears.upto_id, EXCLUDED.upto_id), cleared_at = NOW()
	`, roomID, userID)
	return err
}

// DeleteDirectRoom — «удалить у обоих»: комната и переписка целиком (CASCADE
// снесёт сообщения, реакции, отметки прочтения и очистки).
func (r *chatRepo) DeleteDirectRoom(ctx context.Context, roomID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM chat_rooms WHERE id = $1 AND kind = 'direct'`, roomID)
	return err
}

func (r *chatRepo) ClearPoint(ctx context.Context, userID, roomID int64) (int64, error) {
	var upto int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE((SELECT upto_id FROM chat_clears WHERE room_id = $1 AND user_id = $2), 0)
	`, roomID, userID).Scan(&upto)
	return upto, err
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

func (r *chatRepo) InsertMessage(ctx context.Context, roomID, userID int64, body string, replyToID *int64, media *models.ChatMedia) (*models.ChatMessage, error) {
	var mediaJSON []byte
	if media != nil {
		mediaJSON, _ = json.Marshal(media)
	}
	// INSERT + JOIN автора одним round-trip'ом. reply_to_id валиден только в той
	// же комнате (иначе NULL — защита от ответа на чужую комнату).
	row := r.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO chat_messages (room_id, user_id, body, reply_to_id, media)
			VALUES ($1, $2, $3, (SELECT id FROM chat_messages WHERE id = $4 AND room_id = $1), $5)
			RETURNING id, room_id, user_id, body, deleted, created_at, edited, reply_to_id, media
		)
		SELECT ins.id, ins.room_id, ins.user_id,
		       COALESCE(u.display_name, ''), COALESCE(u.favorite_club, ''),
		       ins.body, ins.deleted, ins.created_at, ins.edited, ins.reply_to_id, ins.media
		FROM ins LEFT JOIN users u ON u.id = ins.user_id
	`, roomID, userID, body, replyToID, mediaJSON)
	return scanMessage(row)
}

func (r *chatRepo) ListMessages(ctx context.Context, roomID, beforeID, sinceID int64, limit int, minID int64) ([]*models.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var (
		query string
		args  []any
		desc  bool // выбрали по убыванию (history) — перевернём в конце
	)
	// minID — очистка «удалить чат у меня»: всё, что не новее minID, скрыто.
	switch {
	case sinceID > 0:
		if sinceID < minID {
			sinceID = minID
		}
		// Catch-up: новее since, по возрастанию.
		query = `SELECT m.id, m.room_id, m.user_id, COALESCE(u.display_name,''),
			        COALESCE(u.favorite_club,''), m.body, m.deleted, m.created_at, m.edited, m.reply_to_id, m.media
			FROM chat_messages m LEFT JOIN users u ON u.id = m.user_id
			WHERE m.room_id = $1 AND m.id > $2 ORDER BY m.id ASC LIMIT $3`
		args = []any{roomID, sinceID, limit}
	case beforeID > 0:
		query = `SELECT m.id, m.room_id, m.user_id, COALESCE(u.display_name,''),
			        COALESCE(u.favorite_club,''), m.body, m.deleted, m.created_at, m.edited, m.reply_to_id, m.media
			FROM chat_messages m LEFT JOIN users u ON u.id = m.user_id
			WHERE m.room_id = $1 AND m.id < $2 AND m.id > $4 ORDER BY m.id DESC LIMIT $3`
		args = []any{roomID, beforeID, limit, minID}
		desc = true
	default:
		query = `SELECT m.id, m.room_id, m.user_id, COALESCE(u.display_name,''),
			        COALESCE(u.favorite_club,''), m.body, m.deleted, m.created_at, m.edited, m.reply_to_id, m.media
			FROM chat_messages m LEFT JOIN users u ON u.id = m.user_id
			WHERE m.room_id = $1 AND m.id > $3 ORDER BY m.id DESC LIMIT $2`
		args = []any{roomID, limit, minID}
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
			RETURNING id, room_id, user_id, body, deleted, created_at, edited, reply_to_id, media
		)
		SELECT upd.id, upd.room_id, upd.user_id,
		       COALESCE(u.display_name,''), COALESCE(u.favorite_club,''),
		       upd.body, upd.deleted, upd.created_at, upd.edited, upd.reply_to_id, upd.media
		FROM upd LEFT JOIN users u ON u.id = upd.user_id
	`, messageID)
	return scanMessage(row)
}

// AddReaction ставит реакцию по правилу «одна реакция на пользователя»:
// прежняя (другая) реакция пользователя на это сообщение снимается и
// возвращается как prev, чтобы сервис разослал её снятие.
func (r *chatRepo) AddReaction(ctx context.Context, messageID, userID int64, emoji string) (prev string, inserted bool, err error) {
	err = r.db.QueryRow(ctx, `
		WITH del AS (
			DELETE FROM chat_reactions
			WHERE message_id = $1 AND user_id = $2 AND emoji <> $3
			RETURNING emoji
		), ins AS (
			INSERT INTO chat_reactions (message_id, user_id, emoji)
			VALUES ($1, $2, $3) ON CONFLICT DO NOTHING
			RETURNING emoji
		)
		SELECT COALESCE((SELECT min(emoji) FROM del), ''), EXISTS(SELECT 1 FROM ins)
	`, messageID, userID, emoji).Scan(&prev, &inserted)
	return prev, inserted, err
}

func (r *chatRepo) RemoveReaction(ctx context.Context, messageID, userID int64, emoji string) (bool, error) {
	ct, err := r.db.Exec(ctx, `
		DELETE FROM chat_reactions WHERE message_id = $1 AND user_id = $2 AND emoji = $3
	`, messageID, userID, emoji)
	return ct.RowsAffected() > 0, err
}

func (r *chatRepo) RoomReactions(ctx context.Context, roomID, userID int64) ([]models.ReactionAgg, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cr.message_id, cr.emoji, count(*) AS cnt, bool_or(cr.user_id = $2) AS mine,
		       jsonb_agg(jsonb_build_object(
		           'id', cr.user_id,
		           'name', COALESCE(u.display_name, ''),
		           'club', COALESCE(u.favorite_club, '')
		       ) ORDER BY cr.created_at) AS users
		FROM chat_reactions cr
		JOIN chat_messages m ON m.id = cr.message_id
		LEFT JOIN users u ON u.id = cr.user_id
		WHERE m.room_id = $1
		GROUP BY cr.message_id, cr.emoji
		ORDER BY cr.message_id, min(cr.created_at)
	`, roomID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ReactionAgg
	for rows.Next() {
		var a models.ReactionAgg
		var usersJSON []byte
		if err := rows.Scan(&a.MessageID, &a.Emoji, &a.Count, &a.Mine, &usersJSON); err != nil {
			return nil, err
		}
		if len(usersJSON) > 0 {
			_ = json.Unmarshal(usersJSON, &a.Users)
		}
		// Для аватарок хватает первых пяти — остальное показывается числом.
		if len(a.Users) > 5 {
			a.Users = a.Users[:5]
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UserBrief — имя и клуб пользователя (для аватарки в событии реакции).
func (r *chatRepo) UserBrief(ctx context.Context, userID int64) (name, club string, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(display_name, ''), COALESCE(favorite_club, '') FROM users WHERE id = $1
	`, userID).Scan(&name, &club)
	return name, club, err
}

func (r *chatRepo) MessageMeta(ctx context.Context, messageID int64) (int64, int64, error) {
	var author *int64
	var roomID int64
	err := r.db.QueryRow(ctx, `SELECT user_id, room_id FROM chat_messages WHERE id = $1`, messageID).
		Scan(&author, &roomID)
	if err != nil {
		return 0, 0, err
	}
	if author == nil {
		return 0, roomID, nil
	}
	return *author, roomID, nil
}

func (r *chatRepo) UpdateMessage(ctx context.Context, messageID int64, body string) (*models.ChatMessage, error) {
	row := r.db.QueryRow(ctx, `
		WITH upd AS (
			UPDATE chat_messages SET body = $2, edited = TRUE
			WHERE id = $1 AND NOT deleted
			RETURNING id, room_id, user_id, body, deleted, created_at, edited, reply_to_id, media
		)
		SELECT upd.id, upd.room_id, upd.user_id,
		       COALESCE(u.display_name,''), COALESCE(u.favorite_club,''),
		       upd.body, upd.deleted, upd.created_at, upd.edited, upd.reply_to_id, upd.media
		FROM upd LEFT JOIN users u ON u.id = upd.user_id
	`, messageID, body)
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
	var mediaRaw []byte
	if err := row.Scan(&m.ID, &m.RoomID, &m.UserID, &m.AuthorName, &m.AuthorClub,
		&m.Body, &m.Deleted, &m.CreatedAt, &m.Edited, &m.ReplyToID, &mediaRaw); err != nil {
		return nil, err
	}
	// Удалённые сообщения не отдаём телом/вложением — только пометку.
	if m.Deleted {
		m.Body = ""
		return m, nil
	}
	if len(mediaRaw) > 0 {
		var med models.ChatMedia
		if json.Unmarshal(mediaRaw, &med) == nil && med.URL != "" {
			m.Media = &med
		}
	}
	return m, nil
}
