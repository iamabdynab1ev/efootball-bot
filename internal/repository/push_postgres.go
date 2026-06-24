package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PushSubscription — web-push подписка браузера пользователя.
type PushSubscription struct {
	UserID   int64
	Endpoint string
	P256dh   string
	Auth     string
}

type PushRepository interface {
	Save(ctx context.Context, sub PushSubscription) error
	DeleteByEndpoint(ctx context.Context, endpoint string) error
	GetByUserIDs(ctx context.Context, userIDs []int64) ([]PushSubscription, error)
	GetAll(ctx context.Context) ([]PushSubscription, error)
}

type pushRepo struct {
	db *pgxpool.Pool
}

func NewPushRepository(db *pgxpool.Pool) PushRepository {
	return &pushRepo{db: db}
}

func (r *pushRepo) Save(ctx context.Context, s PushSubscription) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO UPDATE
		  SET user_id = EXCLUDED.user_id,
		      p256dh  = EXCLUDED.p256dh,
		      auth    = EXCLUDED.auth
	`, s.UserID, s.Endpoint, s.P256dh, s.Auth)
	return err
}

func (r *pushRepo) DeleteByEndpoint(ctx context.Context, endpoint string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint)
	return err
}

func (r *pushRepo) GetAll(ctx context.Context) ([]PushSubscription, error) {
	rows, err := r.db.Query(ctx, `SELECT user_id, endpoint, p256dh, auth FROM push_subscriptions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.UserID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (r *pushRepo) GetByUserIDs(ctx context.Context, userIDs []int64) ([]PushSubscription, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT user_id, endpoint, p256dh, auth
		FROM push_subscriptions
		WHERE user_id = ANY($1)
	`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.UserID, &s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
