package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type statsRepo struct{ db *pgxpool.Pool }

func NewStatsRepository(db *pgxpool.Pool) StatsRepository {
	return &statsRepo{db: db}
}

func (r *statsRepo) GetWinRate(ctx context.Context, minGames int) ([]*StatEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.display_name, u.favorite_club, u.rank, u.rating,
		       COALESCE(SUM(lm.wins),   0)::int,
		       COALESCE(SUM(lm.draws),  0)::int,
		       COALESCE(SUM(lm.losses), 0)::int,
		       COALESCE(SUM(lm.wins + lm.draws + lm.losses), 0)::int
		FROM users u
		JOIN league_members lm ON lm.user_id = u.id AND lm.status = 'approved'
		WHERE u.is_banned = false
		GROUP BY u.id, u.display_name, u.favorite_club, u.rank, u.rating
		HAVING SUM(lm.wins + lm.draws + lm.losses) >= $1
		ORDER BY
		  (SUM(lm.wins)::float / NULLIF(SUM(lm.wins + lm.draws + lm.losses), 0)) DESC,
		  SUM(lm.wins) DESC
		LIMIT 50
	`, minGames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*StatEntry
	for rows.Next() {
		e := &StatEntry{}
		if err := rows.Scan(&e.UserID, &e.DisplayName, &e.FavoriteClub, &e.Rank, &e.Rating,
			&e.Wins, &e.Draws, &e.Losses, &e.Played); err != nil {
			return nil, err
		}
		if e.Played > 0 {
			e.WinRate = float64(e.Wins) / float64(e.Played) * 100
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *statsRepo) GetAvgGoals(ctx context.Context, minGames int) ([]*StatEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.display_name, u.favorite_club, u.rank, u.rating,
		       COALESCE(SUM(lm.wins + lm.draws + lm.losses), 0)::int,
		       COALESCE(SUM(lm.goals_for),     0)::int,
		       COALESCE(SUM(lm.goals_against), 0)::int
		FROM users u
		JOIN league_members lm ON lm.user_id = u.id AND lm.status = 'approved'
		WHERE u.is_banned = false
		GROUP BY u.id, u.display_name, u.favorite_club, u.rank, u.rating
		HAVING SUM(lm.wins + lm.draws + lm.losses) >= $1
		ORDER BY
		  (SUM(lm.goals_for)::float / NULLIF(SUM(lm.wins + lm.draws + lm.losses), 0)) DESC
		LIMIT 50
	`, minGames)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*StatEntry
	for rows.Next() {
		e := &StatEntry{}
		if err := rows.Scan(&e.UserID, &e.DisplayName, &e.FavoriteClub, &e.Rank, &e.Rating,
			&e.Played, &e.GoalsFor, &e.GoalsAgainst); err != nil {
			return nil, err
		}
		if e.Played > 0 {
			e.AvgGoals = float64(e.GoalsFor) / float64(e.Played)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *statsRepo) GetTeamPower(ctx context.Context) ([]*StatEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.display_name, u.favorite_club, u.rank, u.rating, u.team_power
		FROM users u
		WHERE u.is_banned = false AND u.team_power > 0
		ORDER BY u.team_power DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*StatEntry
	for rows.Next() {
		e := &StatEntry{}
		if err := rows.Scan(&e.UserID, &e.DisplayName, &e.FavoriteClub, &e.Rank, &e.Rating, &e.TeamPower); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *statsRepo) GetActivity(ctx context.Context) ([]*StatEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT u.id, u.display_name, u.favorite_club, u.rank, u.rating,
		       COALESCE(SUM(lm.wins + lm.draws + lm.losses), 0)::int,
		       COALESCE(SUM(lm.wins),      0)::int,
		       COALESCE(SUM(lm.goals_for), 0)::int
		FROM users u
		JOIN league_members lm ON lm.user_id = u.id AND lm.status = 'approved'
		WHERE u.is_banned = false
		GROUP BY u.id, u.display_name, u.favorite_club, u.rank, u.rating
		HAVING SUM(lm.wins + lm.draws + lm.losses) > 0
		ORDER BY SUM(lm.wins + lm.draws + lm.losses) DESC, SUM(lm.wins) DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*StatEntry
	for rows.Next() {
		e := &StatEntry{}
		if err := rows.Scan(&e.UserID, &e.DisplayName, &e.FavoriteClub, &e.Rank, &e.Rating,
			&e.Played, &e.Wins, &e.GoalsFor); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (r *statsRepo) GetStreaks(ctx context.Context) ([]*StatEntry, error) {
	// Для каждого активного игрока берём последние 30 матчей и считаем
	// текущую серию побед (consecutive W's from the most recent match).
	rows, err := r.db.Query(ctx, `
		WITH active_users AS (
		  SELECT DISTINCT user_id FROM league_members WHERE status = 'approved'
		),
		user_results AS (
		  SELECT
		    au.user_id,
		    CASE
		      WHEN m.home_user_id = au.user_id AND m.home_goals > m.away_goals THEN 'W'
		      WHEN m.away_user_id = au.user_id AND m.away_goals > m.home_goals THEN 'W'
		      WHEN m.home_goals = m.away_goals THEN 'D'
		      ELSE 'L'
		    END AS result,
		    ROW_NUMBER() OVER (PARTITION BY au.user_id ORDER BY m.played_at DESC) AS rn
		  FROM active_users au
		  JOIN matches m ON (m.home_user_id = au.user_id OR m.away_user_id = au.user_id)
		  WHERE m.status = 'confirmed'
		    AND m.home_goals IS NOT NULL AND m.away_goals IS NOT NULL
		),
		with_group AS (
		  SELECT user_id, result, rn,
		    SUM(CASE WHEN result <> 'W' THEN 1 ELSE 0 END)
		      OVER (PARTITION BY user_id ORDER BY rn) AS brk
		  FROM user_results WHERE rn <= 30
		),
		streaks AS (
		  SELECT user_id, COUNT(*)::int AS streak
		  FROM with_group
		  WHERE brk = 0 AND result = 'W'
		  GROUP BY user_id
		)
		SELECT u.id, u.display_name, u.favorite_club, u.rank, u.rating, s.streak
		FROM streaks s
		JOIN users u ON u.id = s.user_id
		WHERE u.is_banned = false AND s.streak >= 2
		ORDER BY s.streak DESC, u.rating DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*StatEntry
	for rows.Next() {
		e := &StatEntry{}
		if err := rows.Scan(&e.UserID, &e.DisplayName, &e.FavoriteClub, &e.Rank, &e.Rating, &e.Streak); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
