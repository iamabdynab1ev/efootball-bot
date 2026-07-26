package repository

import (
	"context"
	"efootball-bot/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type deadlineRepo struct {
	db *pgxpool.Pool
}

func NewDeadlineRepository(db *pgxpool.Pool) DeadlineRepository {
	return &deadlineRepo{db: db}
}

const deadlineCols = `id, league_id, round, stage, deadline, reminder_24h_sent, reminder_1h_sent, processed_at, created_at`

func collectDeadlines(rows pgx.Rows) ([]*models.RoundDeadline, error) {
	defer rows.Close()
	var result []*models.RoundDeadline
	for rows.Next() {
		d := &models.RoundDeadline{}
		if err := rows.Scan(&d.ID, &d.LeagueID, &d.Round, &d.Stage, &d.Deadline,
			&d.Reminder24hSent, &d.Reminder1hSent, &d.ProcessedAt, &d.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *deadlineRepo) SetDeadline(ctx context.Context, leagueID int64, round int, stage string, deadline time.Time) error {
	// Дедлайн в будущем сбрасывает флаги: напоминания и автоматика отработают заново.
	_, err := r.db.Exec(ctx, `
		INSERT INTO round_deadlines (league_id, round, stage, deadline)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (league_id, round, stage) DO UPDATE SET
		  deadline = EXCLUDED.deadline,
		  reminder_24h_sent = CASE WHEN EXCLUDED.deadline > NOW() THEN FALSE ELSE round_deadlines.reminder_24h_sent END,
		  reminder_1h_sent  = CASE WHEN EXCLUDED.deadline > NOW() THEN FALSE ELSE round_deadlines.reminder_1h_sent END,
		  processed_at      = CASE WHEN EXCLUDED.deadline > NOW() THEN NULL ELSE round_deadlines.processed_at END
	`, leagueID, round, stage, deadline)
	return err
}

func (r *deadlineRepo) GetDeadlines(ctx context.Context, leagueID int64) ([]*models.RoundDeadline, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+deadlineCols+`
		FROM round_deadlines WHERE league_id=$1
		ORDER BY (stage != ''), round, deadline
	`, leagueID)
	if err != nil {
		return nil, err
	}
	return collectDeadlines(rows)
}

func (r *deadlineRepo) GetPendingReminders(ctx context.Context, now time.Time) ([]*models.RoundDeadline, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+deadlineCols+`
		FROM round_deadlines
		WHERE processed_at IS NULL AND (
			(NOT reminder_1h_sent  AND deadline BETWEEN $1 AND $2) OR
			(NOT reminder_24h_sent AND deadline BETWEEN $3 AND $4)
		)
	`,
		now, now.Add(65*time.Minute),
		now.Add(65*time.Minute), now.Add(25*time.Hour),
	)
	if err != nil {
		return nil, err
	}
	return collectDeadlines(rows)
}

func (r *deadlineRepo) MarkReminderSent(ctx context.Context, id int64, is24h bool) error {
	if is24h {
		_, err := r.db.Exec(ctx, `UPDATE round_deadlines SET reminder_24h_sent=TRUE WHERE id=$1`, id)
		return err
	}
	_, err := r.db.Exec(ctx, `UPDATE round_deadlines SET reminder_1h_sent=TRUE WHERE id=$1`, id)
	return err
}

func (r *deadlineRepo) DeleteDeadline(ctx context.Context, leagueID int64, round int, stage string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM round_deadlines WHERE league_id=$1 AND round=$2 AND stage=$3`,
		leagueID, round, stage)
	return err
}

func (r *deadlineRepo) DueUnprocessed(ctx context.Context, now time.Time) ([]*models.RoundDeadline, error) {
	rows, err := r.db.Query(ctx, `
		SELECT d.id, d.league_id, d.round, d.stage, d.deadline,
		       d.reminder_24h_sent, d.reminder_1h_sent, d.processed_at, d.created_at
		FROM round_deadlines d
		JOIN leagues l ON l.id = d.league_id AND l.status = 'active'
		WHERE d.deadline <= $1 AND d.processed_at IS NULL
		ORDER BY d.deadline
	`, now)
	if err != nil {
		return nil, err
	}
	return collectDeadlines(rows)
}

func (r *deadlineRepo) MarkProcessed(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE round_deadlines SET processed_at=NOW() WHERE id=$1`, id)
	return err
}
