package repository

import (
	"context"
	"efootball-bot/internal/models"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type deadlineRepo struct {
	db *pgxpool.Pool
}

func NewDeadlineRepository(db *pgxpool.Pool) DeadlineRepository {
	return &deadlineRepo{db: db}
}

func (r *deadlineRepo) SetDeadline(ctx context.Context, leagueID int64, round int, deadline time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO round_deadlines (league_id, round, deadline)
		VALUES ($1, $2, $3)
		ON CONFLICT (league_id, round) DO UPDATE
		  SET deadline=$3, reminder_24h_sent=FALSE, reminder_1h_sent=FALSE
	`, leagueID, round, deadline)
	return err
}

func (r *deadlineRepo) GetDeadlines(ctx context.Context, leagueID int64) ([]*models.RoundDeadline, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, league_id, round, deadline, reminder_24h_sent, reminder_1h_sent, created_at
		FROM round_deadlines WHERE league_id=$1 ORDER BY round ASC
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.RoundDeadline
	for rows.Next() {
		d := &models.RoundDeadline{}
		if err := rows.Scan(&d.ID, &d.LeagueID, &d.Round, &d.Deadline,
			&d.Reminder24hSent, &d.Reminder1hSent, &d.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *deadlineRepo) GetPendingReminders(ctx context.Context, now time.Time) ([]*models.RoundDeadline, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, league_id, round, deadline, reminder_24h_sent, reminder_1h_sent, created_at
		FROM round_deadlines
		WHERE (
			(NOT reminder_1h_sent  AND deadline BETWEEN $1 AND $2) OR
			(NOT reminder_24h_sent AND deadline BETWEEN $3 AND $4)
		)
	`,
		now.Add(55*time.Minute), now.Add(65*time.Minute),
		now.Add(23*time.Hour), now.Add(25*time.Hour),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.RoundDeadline
	for rows.Next() {
		d := &models.RoundDeadline{}
		if err := rows.Scan(&d.ID, &d.LeagueID, &d.Round, &d.Deadline,
			&d.Reminder24hSent, &d.Reminder1hSent, &d.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *deadlineRepo) MarkReminderSent(ctx context.Context, id int64, is24h bool) error {
	if is24h {
		_, err := r.db.Exec(ctx, `UPDATE round_deadlines SET reminder_24h_sent=TRUE WHERE id=$1`, id)
		return err
	}
	_, err := r.db.Exec(ctx, `UPDATE round_deadlines SET reminder_1h_sent=TRUE WHERE id=$1`, id)
	return err
}

func (r *deadlineRepo) DeleteDeadline(ctx context.Context, leagueID int64, round int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM round_deadlines WHERE league_id=$1 AND round=$2`, leagueID, round)
	return err
}
