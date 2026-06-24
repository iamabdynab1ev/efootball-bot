package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsRepository — простое key-value хранилище настроек приложения
// (контакты поддержки и т.п.).
type SettingsRepository interface {
	GetMany(ctx context.Context, keys []string) (map[string]string, error)
	Set(ctx context.Context, key, value string) error
}

type settingsRepo struct {
	db *pgxpool.Pool
}

func NewSettingsRepository(db *pgxpool.Pool) SettingsRepository {
	return &settingsRepo{db: db}
}

func (r *settingsRepo) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(ctx, `SELECT key, value FROM app_settings WHERE key = ANY($1)`, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

func (r *settingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`, key, value)
	return err
}
