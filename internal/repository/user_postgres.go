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
	       favorite_club,
	       google_id, email, created_at, updated_at
	FROM users`

func scanUser(row pgx.Row) (*models.User, error) {
	u := &models.User{}
	err := row.Scan(
		&u.ID, &u.TelegramID, &u.DisplayName, &u.Username, &u.IsBanned,
		&u.Rating, &u.TeamPower, &u.Rank, &u.Language,
		&u.FavoriteClub,
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

func (r *userRepo) UpdateFavoriteClub(ctx context.Context, userID int64, clubID string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET favorite_club=$1, updated_at=NOW() WHERE id=$2`, clubID, userID)
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
		          favorite_club,
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
		          favorite_club,
		          google_id, email, created_at, updated_at
	`, googleID, email, displayName)
	return scanUser(row)
}

// GetAllByRating — все игроки отсортированные по рейтингу (для страницы рейтинга).
func (r *userRepo) GetAllByRating(ctx context.Context, limit int) ([]*models.User, error) {
	rows, err := r.db.Query(ctx, userSelect+`
		WHERE is_banned=false
		  AND id NOT IN (SELECT user_id FROM admins WHERE role='super_admin')
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
			&u.FavoriteClub,
			&u.GoogleID, &u.Email, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// GenerateLinkCode — генерирует 6-значный код для привязки Telegram.
// DeleteUser полностью удаляет пользователя и все связанные данные. Часть
// таблиц удаляется каскадом при удалении users (league_members, admins,
// admin_credentials, user_achievements), остальные — явно (нет ON DELETE).
// Возвращает id затронутых лиг для пересчёта турнирных таблиц.
func (r *userRepo) DeleteUser(ctx context.Context, userID int64) ([]int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Лиги, которые надо пересчитать после удаления (где есть матчи/членство).
	var leagues []int64
	rows, err := tx.Query(ctx, `
		SELECT DISTINCT league_id FROM (
			SELECT league_id FROM league_members WHERE user_id=$1
			UNION
			SELECT league_id FROM matches WHERE home_user_id=$1 OR away_user_id=$1
		) t WHERE league_id IS NOT NULL
	`, userID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var lid int64
		if err := rows.Scan(&lid); err != nil {
			rows.Close()
			return nil, err
		}
		leagues = append(leagues, lid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Порядок важен: сначала то, что ссылается на матчи/пользователя без каскада.
	steps := []string{
		`DELETE FROM disputes WHERE reported_by=$1
		   OR match_id IN (SELECT id FROM matches WHERE home_user_id=$1 OR away_user_id=$1)`,
		`DELETE FROM bracket_slots WHERE home_user_id=$1 OR away_user_id=$1 OR winner_user_id=$1`,
		`DELETE FROM de_nodes WHERE home_user_id=$1 OR away_user_id=$1 OR winner_user_id=$1`,
		`DELETE FROM season_awards WHERE user_id=$1`,
		`DELETE FROM matches WHERE home_user_id=$1 OR away_user_id=$1`,
		// users: каскадом уйдут league_members, admins, admin_credentials, user_achievements
		`DELETE FROM users WHERE id=$1`,
	}
	for _, q := range steps {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			return nil, fmt.Errorf("delete user: %w", err)
		}
	}
	return leagues, tx.Commit(ctx)
}

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
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 1. Находим аккаунт по действующему коду.
	var userID int64
	err = tx.QueryRow(ctx, `
		SELECT id FROM users
		WHERE telegram_link_code=$1 AND telegram_link_expires > NOW()
	`, code).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // код неверный или истёк
	}
	if err != nil {
		return nil, err
	}

	// 2. Этот Telegram мог быть привязан к другому аккаунту (например, старому,
	//    созданному ботом). Открепляем его оттуда — telegram_id уникален.
	if _, err := tx.Exec(ctx, `
		UPDATE users SET telegram_id=NULL, updated_at=NOW()
		WHERE telegram_id=$1 AND id<>$2
	`, telegramID, userID); err != nil {
		return nil, err
	}

	// 3. Привязываем Telegram к найденному аккаунту.
	row := tx.QueryRow(ctx, `
		UPDATE users
		SET telegram_id=$2, username=$3,
		    telegram_link_code=NULL, telegram_link_expires=NULL,
		    updated_at=NOW()
		WHERE id=$1
		RETURNING id, COALESCE(telegram_id,0), display_name, username, is_banned,
		          rating, team_power, rank,
		          COALESCE(language,'uz') AS language,
		          google_id, email, created_at, updated_at
	`, userID, telegramID, username)
	user, err := scanUser(row)
	if err != nil {
		return nil, err
	}
	return user, tx.Commit(ctx)
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
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// GetTopScorersAllLeagues возвращает бомбардиров по всем активным лигам за один запрос.
type LeagueWithScorers struct {
	LeagueID   int64
	LeagueName string
	Status     string
	Level      int
	MaxPlayers int
	RoundsType string
	Scorers    []*models.LeagueMember
}

func (r *userRepo) GetTopScorersAllLeagues(ctx context.Context) ([]*LeagueWithScorers, error) {
	rows, err := r.db.Query(ctx, `
		SELECT l.id, l.name, l.status::text, l.level, l.max_players, l.rounds_type,
		       lm.user_id, lm.goals_for,
		       u.display_name, u.rating, u.team_power, u.favorite_club,
		       ROW_NUMBER() OVER (PARTITION BY l.id ORDER BY lm.goals_for DESC) AS rn
		FROM leagues l
		JOIN league_members lm ON lm.league_id = l.id AND lm.status = 'approved' AND lm.goals_for > 0
		JOIN users u ON u.id = lm.user_id
		WHERE l.status != 'archived'
		ORDER BY l.id, lm.goals_for DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leagueMap := map[int64]*LeagueWithScorers{}
	var leagueOrder []int64

	for rows.Next() {
		var leagueID int64
		var leagueName, leagueStatus, roundsType string
		var level, maxPlayers int
		var rn int64
		m := &models.LeagueMember{User: &models.User{}}

		if err := rows.Scan(
			&leagueID, &leagueName, &leagueStatus, &level, &maxPlayers, &roundsType,
			&m.UserID, &m.GoalsFor,
			&m.User.DisplayName, &m.User.Rating, &m.User.TeamPower, &m.User.FavoriteClub,
			&rn,
		); err != nil {
			return nil, err
		}
		if rn > 10 {
			continue
		}
		m.LeagueID = leagueID

		if _, ok := leagueMap[leagueID]; !ok {
			leagueMap[leagueID] = &LeagueWithScorers{
				LeagueID:   leagueID,
				LeagueName: leagueName,
				Status:     leagueStatus,
				Level:      level,
				MaxPlayers: maxPlayers,
				RoundsType: roundsType,
			}
			leagueOrder = append(leagueOrder, leagueID)
		}
		leagueMap[leagueID].Scorers = append(leagueMap[leagueID].Scorers, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]*LeagueWithScorers, 0, len(leagueOrder))
	for _, id := range leagueOrder {
		result = append(result, leagueMap[id])
	}
	return result, nil
}

func (r *userRepo) UpdateRank(ctx context.Context, userID int64, rank string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET rank=$1, updated_at=NOW() WHERE id=$2`, rank, userID)
	return err
}

func (r *userRepo) RecalculateAllRanks(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET rank = CASE
			WHEN pct <= 1  THEN '👑 Легенда'
			WHEN pct <= 5  THEN '💎 Элита'
			WHEN pct <= 10 THEN '🔥 Устод'
			WHEN pct <= 20 THEN '⭐️ Профессионал'
			WHEN pct <= 30 THEN '🌟 Тажрибали'
			WHEN pct <= 40 THEN '💪 Ўйинчи'
			WHEN pct <= 55 THEN '🎮 Аматёр'
			WHEN pct <= 70 THEN '⚽ Ҳаваскор'
			WHEN pct <= 85 THEN '🥈 Янги бошловчи'
			ELSE                '🥉 Новичок'
		END
		FROM (
			SELECT id,
			       PERCENT_RANK() OVER (ORDER BY rating DESC) * 100 AS pct
			FROM users
			WHERE is_banned = false
		) ranked
		WHERE users.id = ranked.id AND users.is_banned = false
	`)
	return err
}

func (r *userRepo) GetGlobalStats(ctx context.Context, userID int64) (*models.GlobalStats, error) {
	stats := &models.GlobalStats{}
	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total_matches,
			SUM(CASE
				WHEN (home_user_id=$1 AND home_goals>away_goals) OR (away_user_id=$1 AND away_goals>home_goals)
				THEN 1 ELSE 0 END) AS total_wins,
			SUM(CASE WHEN home_goals=away_goals THEN 1 ELSE 0 END) AS total_draws,
			SUM(CASE
				WHEN (home_user_id=$1 AND home_goals<away_goals) OR (away_user_id=$1 AND away_goals<home_goals)
				THEN 1 ELSE 0 END) AS total_losses,
			SUM(CASE WHEN home_user_id=$1 THEN home_goals ELSE away_goals END) AS total_goals_for,
			SUM(CASE WHEN home_user_id=$1 THEN away_goals ELSE home_goals END) AS total_goals_against
		FROM matches
		WHERE (home_user_id=$1 OR away_user_id=$1)
		  AND status='confirmed'
		  AND home_goals IS NOT NULL
	`, userID).Scan(
		&stats.TotalMatches, &stats.TotalWins, &stats.TotalDraws, &stats.TotalLosses,
		&stats.TotalGoalsFor, &stats.TotalGoalsAgainst,
	)
	if err != nil {
		return stats, err
	}
	return stats, nil
}
