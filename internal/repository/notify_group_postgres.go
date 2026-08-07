package repository

import (
	"context"

	"efootball-bot/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type notifyGroupRepo struct{ db *pgxpool.Pool }

func NewNotifyGroupRepository(db *pgxpool.Pool) NotifyGroupRepository { return &notifyGroupRepo{db: db} }

// Upsert добавляет/обновляет группу по (channel, chat_id): повторный /connect в
// той же группе не плодит дублей, а обновляет название и снова включает её.
func (r *notifyGroupRepo) Upsert(ctx context.Context, channel, chatID, title string) (*models.NotifyGroup, error) {
	g := &models.NotifyGroup{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO notify_groups (channel, chat_id, title)
		VALUES ($1, $2, $3)
		ON CONFLICT (channel, chat_id) DO UPDATE
		  SET title = COALESCE(NULLIF(EXCLUDED.title, ''), notify_groups.title),
		      enabled = TRUE
		RETURNING id, channel, chat_id, title, enabled, created_at
	`, channel, chatID, title).Scan(&g.ID, &g.Channel, &g.ChatID, &g.Title, &g.Enabled, &g.CreatedAt)
	return g, err
}

func (r *notifyGroupRepo) list(ctx context.Context, onlyEnabled bool) ([]*models.NotifyGroup, error) {
	q := `SELECT id, channel, chat_id, title, enabled, created_at FROM notify_groups`
	if onlyEnabled {
		q += ` WHERE enabled`
	}
	q += ` ORDER BY created_at`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*models.NotifyGroup{}
	for rows.Next() {
		g := &models.NotifyGroup{}
		if err := rows.Scan(&g.ID, &g.Channel, &g.ChatID, &g.Title, &g.Enabled, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *notifyGroupRepo) List(ctx context.Context) ([]*models.NotifyGroup, error) {
	return r.list(ctx, false)
}
func (r *notifyGroupRepo) ListEnabled(ctx context.Context) ([]*models.NotifyGroup, error) {
	return r.list(ctx, true)
}

func (r *notifyGroupRepo) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	_, err := r.db.Exec(ctx, `UPDATE notify_groups SET enabled = $2 WHERE id = $1`, id, enabled)
	return err
}

func (r *notifyGroupRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM notify_groups WHERE id = $1`, id)
	return err
}

// DeleteByChat убирает группу по каналу+chat_id (для /disconnect в самой группе).
func (r *notifyGroupRepo) DeleteByChat(ctx context.Context, channel, chatID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM notify_groups WHERE channel = $1 AND chat_id = $2`, channel, chatID)
	return err
}
