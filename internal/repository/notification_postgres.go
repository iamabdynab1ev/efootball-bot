package repository

import (
	"context"
	"efootball-bot/internal/models"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository interface {
	// CreateBatch вставляет пачку уведомлений одним multi-row INSERT и проставляет
	// id/created_at обратно (для живой публикации). Часто адресатов несколько
	// (оба игрока матча), поэтому батч — основной путь.
	CreateBatch(ctx context.Context, items []*models.Notification) error
	ListByUser(ctx context.Context, userID, beforeID int64, limit int) ([]*models.Notification, error)
	CountUnread(ctx context.Context, userID int64) (int, error)
	// MarkRead помечает прочитанными указанные id пользователя (или все, если ids пуст).
	MarkRead(ctx context.Context, userID int64, ids []int64) error
	Prune(ctx context.Context, keepDays int) (int64, error)
}

type notificationRepo struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) NotificationRepository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) CreateBatch(ctx context.Context, items []*models.Notification) error {
	if len(items) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO notifications (user_id, type, title, body, link) VALUES `)
	args := make([]any, 0, len(items)*5)
	for i, n := range items {
		if i > 0 {
			b.WriteByte(',')
		}
		k := i * 5
		b.WriteString("($" + strconv.Itoa(k+1) + ",$" + strconv.Itoa(k+2) + ",$" + strconv.Itoa(k+3) +
			",$" + strconv.Itoa(k+4) + ",$" + strconv.Itoa(k+5) + ")")
		args = append(args, n.UserID, n.Type, n.Title, nullStr(n.Body), nullStr(n.Link))
	}
	b.WriteString(" RETURNING id, created_at, read")

	rows, err := r.db.Query(ctx, b.String(), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		if err := rows.Scan(&items[i].ID, &items[i].CreatedAt, &items[i].Read); err != nil {
			return err
		}
		i++
	}
	return rows.Err()
}

func (r *notificationRepo) ListByUser(ctx context.Context, userID, beforeID int64, limit int) ([]*models.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var b strings.Builder
	b.WriteString(`SELECT id, user_id, type, title, body, link, read, created_at
		FROM notifications WHERE user_id = $1`)
	args := []any{userID}
	if beforeID > 0 {
		args = append(args, beforeID)
		b.WriteString(" AND id < $" + strconv.Itoa(len(args)))
	}
	args = append(args, limit)
	b.WriteString(" ORDER BY id DESC LIMIT $" + strconv.Itoa(len(args)))

	rows, err := r.db.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Notification
	for rows.Next() {
		n := &models.Notification{}
		var body, link *string
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &body, &link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		if body != nil {
			n.Body = *body
		}
		if link != nil {
			n.Link = *link
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *notificationRepo) CountUnread(ctx context.Context, userID int64) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = FALSE`, userID).Scan(&n)
	return n, err
}

func (r *notificationRepo) MarkRead(ctx context.Context, userID int64, ids []int64) error {
	if len(ids) == 0 {
		// Пустой список = пометить все непрочитанные пользователя.
		_, err := r.db.Exec(ctx,
			`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND read = FALSE`, userID)
		return err
	}
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET read = TRUE WHERE user_id = $1 AND id = ANY($2) AND read = FALSE`,
		userID, ids)
	return err
}

func (r *notificationRepo) Prune(ctx context.Context, keepDays int) (int64, error) {
	if keepDays <= 0 {
		return 0, nil
	}
	// Удаляем только прочитанные старше срока — непрочитанные не теряем.
	tag, err := r.db.Exec(ctx,
		`DELETE FROM notifications WHERE read = TRUE AND created_at < NOW() - make_interval(days => $1)`,
		keepDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
