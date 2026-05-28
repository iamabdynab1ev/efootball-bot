package repository

import (
	"context"
	"crypto/rand"
	"efootball-bot/internal/models"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

const userSelect = `
	SELECT id, COALESCE(telegram_id,0), display_name, username, is_banned,
	       rating, team_power, rank,
	       COALESCE(language,'uz') AS language,
	       google_id, email, created_at, updated_at
	FROM users`

func scanUser(row pgx.Row) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(
		&u.ID, &u.TelegramID, &u.DisplayName, &u.Username, &u.IsBanned,
		&u.Rating, &u.TeamPower, &u.Rank, &u.Language,
		&u.GoogleID, &u.Email, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *userRepo) UpdateLanguage(ctx context.Context, userID int64, lang string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET language=$1, updated_at=NOW() WHERE id=$2`, lang, userID)
	return err
}

func (r *userRepo) Create(ctx context.Context, telegramID int64, displayName string, username *string) (*models.User, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO users (telegram_id, display_name, username)
		VALUES ($1, $2, $3)
		ON CONFLICT (telegram_id) DO UPDATE
		  SET display_name=EXCLUDED.display_name,
		      username=EXCLUDED.username,
		      updated_at=NOW()
		RETURNING id, COALESCE(telegram_id,0), display_name, username, is_banned,
		          rating, team_power, rank,
		          COALESCE(language,'uz') AS language,
		          google_id, email, created_at, updated_at
	`, telegramID, displayName, username)
	return scanUser(row)
}

func (r *userRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*models.User, error) {
	row := r.db.QueryRow(ctx, userSelect+` WHERE telegram_id=$1`, telegramID)
	return scanUser(row)
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	row := r.db.QueryRow(ctx, userSelect+` WHERE id=$1`, id)
	return scanUser(row)
}

func (r *userRepo) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	row := r.db.QueryRow(ctx, userSelect+` WHERE google_id=$1`, googleID)
	return scanUser(row)
}

// UpsertByGoogle — создаёт или находит пользователя по google_id.
func (r *userRepo) UpsertByGoogle(ctx context.Context, googleID, email, displayName string) (*models.User, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO users (google_id, email, display_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (google_id) DO UPDATE
		  SET email=EXCLUDED.email,
		      updated_at=NOW()
		RETURNING id, COALESCE(telegram_id,0), display_name, username, is_banned,
		          rating, team_power, rank,
		          COALESCE(language,'uz') AS language,
		          google_id, email, created_at, updated_at
	`, googleID, email, displayName)
	return scanUser(row)
}

// GetAllByRating — все игроки отсортированные по рейтингу (для страницы рейтинга).
func (r *userRepo) GetAllByRating(ctx context.Context, limit int) ([]*models.User, error) {
	rows, err := r.db.Query(ctx, userSelect+`
		WHERE is_banned=false
		ORDER BY rating DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(
			&u.ID, &u.TelegramID, &u.DisplayName, &u.Username, &u.IsBanned,
			&u.Rating, &u.TeamPower, &u.Rank, &u.Language,
			&u.GoogleID, &u.Email, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// GenerateLinkCode — генерирует 6-значный код для привязки Telegram.
func (r *userRepo) GenerateLinkCode(ctx context.Context, userID int64) (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2]))
	code = code[:6]

	expires := time.Now().Add(10 * time.Minute)
	_, err := r.db.Exec(ctx, `
		UPDATE users SET telegram_link_code=$1, telegram_link_expires=$2, updated_at=NOW()
		WHERE id=$3
	`, code, expires, userID)
	return code, err
}

// LinkTelegramByCode — бот вызывает этот метод когда пользователь отправляет /link CODE.
func (r *userRepo) LinkTelegramByCode(ctx context.Context, code string, telegramID int64, username *string) (*models.User, error) {
	row := r.db.QueryRow(ctx, `
		UPDATE users
		SET telegram_id=$2, username=$3,
		    telegram_link_code=NULL, telegram_link_expires=NULL,
		    updated_at=NOW()
		WHERE telegram_link_code=$1
		  AND telegram_link_expires > NOW()
		RETURNING id, COALESCE(telegram_id,0), display_name, username, is_banned,
		          rating, team_power, rank,
		          COALESCE(language,'uz') AS language,
		          google_id, email, created_at, updated_at
	`, code, telegramID, username)
	return scanUser(row)
}

func (r *userRepo) UpdateDisplayName(ctx context.Context, id int64, name string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET display_name=$1, updated_at=NOW() WHERE id=$2`, name, id)
	return err
}

func (r *userRepo) UpdateRating(ctx context.Context, userID int64, newRating int) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET rating=$1, updated_at=NOW() WHERE id=$2`, newRating, userID)
	return err
}

func (r *userRepo) UpdateTeamPower(ctx context.Context, userID int64, tp int) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET team_power=$1, updated_at=NOW() WHERE id=$2`, tp, userID)
	return err
}

func (r *userRepo) ResetAllRatings(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET rating=1000, rank='🥉 Новичок', updated_at=NOW() WHERE is_banned=false`)
	return err
}

func (r *userRepo) GetTopScorers(ctx context.Context, leagueID int64) ([]*models.LeagueMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.id, lm.league_id, lm.user_id, lm.goals_for,
		       u.display_name, u.rating, u.team_power
		FROM league_members lm
		JOIN users u ON u.id = lm.user_id
		WHERE lm.league_id=$1 AND lm.status='approved'
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
	_, err := r.db.Exec(ctx, `UPDATE users SET rank=$1, updated_at=NOW() WHERE id=$2`, rank, userID)
	return err
}

func (r *userRepo) RecalculateAllRanks(ctx context.Context) error {
	rows, err := r.db.Query(ctx, `SELECT id, rating FROM users WHERE is_banned=false ORDER BY rating DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type ur struct {
		id     int64
		rating int
	}
	var users []ur
	for rows.Next() {
		var u ur
		if err := rows.Scan(&u.id, &u.rating); err != nil {
			continue
		}
		users = append(users, u)
	}
	rows.Close()

	if len(users) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	total := len(users)
	for i, u := range users {
		pct := float64(i+1) / float64(total) * 100
		_, err := tx.Exec(ctx, `UPDATE users SET rank=$1 WHERE id=$2`, calcRank(pct), u.id)
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
