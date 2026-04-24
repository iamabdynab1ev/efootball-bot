// user_postgres.go - Postgres'да фойдаланувчиларни бошқариш учун репозиторий
package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}
func (r *userRepo) UpdateLanguage(ctx context.Context, userID int64, lang string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET language = $1, updated_at = NOW() WHERE id = $2
	`, lang, userID)
	return err
}
func (r *userRepo) Create(ctx context.Context, telegramID int64, displayName string, username *string) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (telegram_id, display_name, username)
		VALUES ($1, $2, $3)
		ON CONFLICT (telegram_id) DO UPDATE
		  SET display_name = EXCLUDED.display_name,
		      username     = EXCLUDED.username,
		      updated_at   = NOW()
		RETURNING id, telegram_id, display_name, username, is_banned,
          rating, team_power, rank,
          COALESCE(language, 'uz') as language,
          created_at, updated_at
	`, telegramID, displayName, username).Scan(
		&u.ID, &u.TelegramID, &u.DisplayName, &u.Username, &u.IsBanned,
		&u.Rating, &u.TeamPower, &u.Rank, &u.Language, &u.CreatedAt, &u.UpdatedAt,
	)
	return u, err
}

func (r *userRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, telegram_id, display_name, username, is_banned,
       rating, team_power, rank, 
       COALESCE(language, 'uz') as language,
       created_at, updated_at
FROM users WHERE telegram_id = $1
	`, telegramID).Scan(
		&u.ID, &u.TelegramID, &u.DisplayName, &u.Username, &u.IsBanned,
		&u.Rating, &u.TeamPower, &u.Rank, &u.Language, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	u := &models.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, telegram_id, display_name, username, is_banned,
       rating, team_power, rank,
       COALESCE(language, 'uz') as language,
       created_at, updated_at
FROM users WHERE id = $1	
	`, id).Scan(
		&u.ID, &u.TelegramID, &u.DisplayName, &u.Username, &u.IsBanned,
		&u.Rating, &u.TeamPower, &u.Rank, &u.Language, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *userRepo) UpdateDisplayName(ctx context.Context, id int64, name string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET display_name=$1, updated_at=NOW() WHERE id=$2
	`, name, id)
	return err
}

func (r *userRepo) UpdateRating(ctx context.Context, userID int64, newRating int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET rating = $1, updated_at = NOW() WHERE id = $2
	`, newRating, userID)
	return err
}

func (r *userRepo) UpdateTeamPower(ctx context.Context, userID int64, tp int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET team_power = $1, updated_at = NOW() WHERE id = $2
	`, tp, userID)
	return err
}
func (r *userRepo) ResetAllRatings(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users 
		SET rating = 1000, rank = '🥉 Новичок', updated_at = NOW()
		WHERE is_banned = false
	`)
	return err
}
func (r *userRepo) GetTopScorers(ctx context.Context, leagueID int64) ([]*models.LeagueMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.id, lm.league_id, lm.user_id, lm.goals_for,
		       u.display_name, u.rating, u.team_power
		FROM league_members lm
		JOIN users u ON u.id = lm.user_id
		WHERE lm.league_id = $1 AND lm.status = 'approved'
		ORDER BY lm.goals_for DESC
		LIMIT 20
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{User: &models.User{}}
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &m.GoalsFor,
			&m.User.DisplayName, &m.User.Rating, &m.User.TeamPower,
		); err != nil {
			continue
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
func (r *userRepo) UpdateRank(ctx context.Context, userID int64, rank string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET rank = $1, updated_at = NOW() WHERE id = $2
	`, rank, userID)
	return err
}

func (r *userRepo) RecalculateAllRanks(ctx context.Context) error {
	// Барча ўйинчиларни рейтинг бўйича оламиз
	rows, err := r.db.Query(ctx, `
		SELECT id, rating FROM users 
		WHERE is_banned = false
		ORDER BY rating DESC
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type userRating struct {
		id     int64
		rating int
	}

	var users []userRating
	for rows.Next() {
		var u userRating
		if err := rows.Scan(&u.id, &u.rating); err != nil {
			continue
		}
		users = append(users, u)
	}
	rows.Close()

	total := len(users)
	if total == 0 {
		return nil
	}

	// Ҳар бир ўйинчига статус берамиз
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for i, u := range users {
		percent := float64(i+1) / float64(total) * 100
		rank := calcRank(percent)
		_, err := tx.Exec(ctx, `
			UPDATE users SET rank = $1 WHERE id = $2
		`, rank, u.id)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func calcRank(topPercent float64) string {
	switch {
	case topPercent <= 1:
		return "👑 Легенда"
	case topPercent <= 5:
		return "💎 Элита"
	case topPercent <= 10:
		return "🔥 Устод"
	case topPercent <= 20:
		return "⭐️ Профессионал"
	case topPercent <= 30:
		return "🌟 Тажрибали"
	case topPercent <= 40:
		return "💪 Ўйинчи"
	case topPercent <= 55:
		return "🎮 Аматёр"
	case topPercent <= 70:
		return "⚽ Ҳаваскор"
	case topPercent <= 85:
		return "🥈 Янги бошловчи"
	default:
		return "🥉 Новичок"
	}
}
