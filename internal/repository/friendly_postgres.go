package repository

import (
	"context"
	"errors"

	"efootball-bot/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrFriendlyActiveExists — между парой игроков уже есть незавершённый вызов.
var ErrFriendlyActiveExists = errors.New("между вами уже есть незавершённый товарищеский матч")

// staleFriendlyCond — какие незавершённые матчи считаем зависшими (TTL):
// вызов без ответа — 48 ч, принятый но не сыгранный — 7 дней, счёт без
// подтверждения соперника — 48 ч. По истечении статус сбрасывается в
// 'expired', и пара снова может вызывать друг друга.
const staleFriendlyCond = `(
	   (status = 'pending'       AND updated_at < NOW() - INTERVAL '48 hours')
	OR (status = 'accepted'      AND updated_at < NOW() - INTERVAL '7 days')
	OR (status = 'score_claimed' AND updated_at < NOW() - INTERVAL '48 hours')
)`

type FriendlyRepository interface {
	Create(ctx context.Context, challengerID, opponentID int64) (*models.Friendly, error)
	Get(ctx context.Context, id int64) (*models.Friendly, error)
	SetStatus(ctx context.Context, id int64, from, to string) (bool, error)
	ClaimScore(ctx context.Context, id, byUser int64, challengerGoals, opponentGoals int16) (bool, error)
	Confirm(ctx context.Context, id int64) (bool, error)
	ListForUser(ctx context.Context, userID int64, limit int) ([]*models.Friendly, error)
	ExpireStale(ctx context.Context) ([]models.FriendlyRef, error)
}

type friendlyRepo struct {
	db *pgxpool.Pool
}

func NewFriendlyRepository(db *pgxpool.Pool) FriendlyRepository {
	return &friendlyRepo{db: db}
}

const friendlySelect = `
	SELECT f.id, f.challenger_id, f.opponent_id, f.status,
	       f.challenger_goals, f.opponent_goals, f.claimed_by, f.created_at, f.updated_at,
	       cu.display_name, COALESCE(cu.favorite_club, ''),
	       ou.display_name, COALESCE(ou.favorite_club, '')
	FROM friendlies f
	JOIN users cu ON cu.id = f.challenger_id
	JOIN users ou ON ou.id = f.opponent_id
`

func scanFriendly(row pgx.Row) (*models.Friendly, error) {
	f := &models.Friendly{}
	err := row.Scan(
		&f.ID, &f.ChallengerID, &f.OpponentID, &f.Status,
		&f.ChallengerGoals, &f.OpponentGoals, &f.ClaimedBy, &f.CreatedAt, &f.UpdatedAt,
		&f.ChallengerName, &f.ChallengerClub, &f.OpponentName, &f.OpponentClub,
	)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (r *friendlyRepo) Create(ctx context.Context, challengerID, opponentID int64) (*models.Friendly, error) {
	// Ленивое истечение для этой пары: зависший старый матч не должен
	// блокировать новый вызов, даже если фоновая очистка ещё не прошла.
	_, _ = r.db.Exec(ctx, `
		UPDATE friendlies SET status = 'expired', updated_at = NOW()
		WHERE LEAST(challenger_id, opponent_id) = LEAST($1::bigint, $2::bigint)
		  AND GREATEST(challenger_id, opponent_id) = GREATEST($1::bigint, $2::bigint)
		  AND `+staleFriendlyCond, challengerID, opponentID)

	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO friendlies (challenger_id, opponent_id) VALUES ($1, $2) RETURNING id
	`, challengerID, opponentID).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // uq_friendlies_active_pair
			return nil, ErrFriendlyActiveExists
		}
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *friendlyRepo) Get(ctx context.Context, id int64) (*models.Friendly, error) {
	return scanFriendly(r.db.QueryRow(ctx, friendlySelect+` WHERE f.id = $1`, id))
}

// SetStatus — переход статуса с проверкой текущего (оптимистичная блокировка).
func (r *friendlyRepo) SetStatus(ctx context.Context, id int64, from, to string) (bool, error) {
	ct, err := r.db.Exec(ctx, `
		UPDATE friendlies SET status = $3, updated_at = NOW() WHERE id = $1 AND status = $2
	`, id, from, to)
	return ct.RowsAffected() > 0, err
}

// ClaimScore — внесение счёта одним из участников (из статуса accepted или
// повторно из score_claimed — например, после исправления).
func (r *friendlyRepo) ClaimScore(ctx context.Context, id, byUser int64, challengerGoals, opponentGoals int16) (bool, error) {
	ct, err := r.db.Exec(ctx, `
		UPDATE friendlies
		SET status = 'score_claimed', challenger_goals = $2, opponent_goals = $3,
		    claimed_by = $4, updated_at = NOW()
		WHERE id = $1 AND status IN ('accepted', 'score_claimed')
	`, id, challengerGoals, opponentGoals, byUser)
	return ct.RowsAffected() > 0, err
}

func (r *friendlyRepo) Confirm(ctx context.Context, id int64) (bool, error) {
	return r.SetStatus(ctx, id, "score_claimed", "confirmed")
}

// ExpireStale — фоновая очистка: переводит зависшие матчи в 'expired' и
// возвращает участников, чтобы вызывающий мог их уведомить.
func (r *friendlyRepo) ExpireStale(ctx context.Context) ([]models.FriendlyRef, error) {
	rows, err := r.db.Query(ctx, `
		UPDATE friendlies SET status = 'expired', updated_at = NOW()
		WHERE `+staleFriendlyCond+`
		RETURNING id, challenger_id, opponent_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.FriendlyRef
	for rows.Next() {
		var ref models.FriendlyRef
		if err := rows.Scan(&ref.ID, &ref.ChallengerID, &ref.OpponentID); err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, rows.Err()
}

func (r *friendlyRepo) ListForUser(ctx context.Context, userID int64, limit int) ([]*models.Friendly, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, friendlySelect+`
		WHERE (f.challenger_id = $1 OR f.opponent_id = $1) AND f.status <> 'cancelled'
		ORDER BY f.updated_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.Friendly
	for rows.Next() {
		f, err := scanFriendly(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
