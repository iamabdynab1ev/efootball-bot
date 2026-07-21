package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PredictionRepository — прогнозы счёта на матчи (виртуальные очки).
type PredictionRepository interface {
	// Upsert сохраняет/обновляет прогноз игрока на матч.
	Upsert(ctx context.Context, matchID, userID int64, homeGoals, awayGoals int16) error
	// MineByLeague — прогнозы игрока по всем матчам лиги: match_id → (h, a, points).
	MineByLeague(ctx context.Context, leagueID, userID int64) ([]*PredictionRow, error)
	// ScoreMatch начисляет очки всем прогнозам подтверждённого матча
	// (5 — точный счёт, 3 — верная разница, 1 — верный исход, 0 — мимо).
	// Возвращает авторов точных прогнозов (для поздравления).
	ScoreMatch(ctx context.Context, matchID int64, homeGoals, awayGoals int16) (exactUserIDs []int64, err error)
	// Leaderboard — таблица прогнозистов лиги.
	Leaderboard(ctx context.Context, leagueID int64) ([]*PredictorRow, error)
	// ByMatch — прогнозы матча с именами (вскрытие после подтверждения).
	ByMatch(ctx context.Context, matchID int64) ([]*PredictionRow, error)
	// CountForMatch — сколько прогнозов уже поставлено на матч.
	CountForMatch(ctx context.Context, matchID int64) (int, error)
	// SeasonPoints — сумма очков прогнозов игроков по лигам сезона («Оракул»).
	SeasonPoints(ctx context.Context, seasonID int64) (map[int64]int, error)
}

// PredictionRow — прогноз с контекстом.
type PredictionRow struct {
	MatchID     int64
	UserID      int64
	DisplayName string
	Club        string
	HomeGoals   int16
	AwayGoals   int16
	Points      *int16
}

// PredictorRow — строка таблицы прогнозистов.
type PredictorRow struct {
	UserID      int64
	DisplayName string
	Club        string
	Points      int
	Exact       int // точных счетов
	Total       int // оценённых прогнозов
}

type predictionRepo struct{ db *pgxpool.Pool }

func NewPredictionRepository(db *pgxpool.Pool) PredictionRepository {
	return &predictionRepo{db: db}
}

func (r *predictionRepo) Upsert(ctx context.Context, matchID, userID int64, homeGoals, awayGoals int16) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO predictions (match_id, user_id, home_goals, away_goals)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (match_id, user_id) DO UPDATE SET
		  home_goals = EXCLUDED.home_goals,
		  away_goals = EXCLUDED.away_goals,
		  updated_at = NOW()
	`, matchID, userID, homeGoals, awayGoals)
	return err
}

func (r *predictionRepo) MineByLeague(ctx context.Context, leagueID, userID int64) ([]*PredictionRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.match_id, p.user_id, '', '', p.home_goals, p.away_goals, p.points
		FROM predictions p
		JOIN matches m ON m.id = p.match_id
		WHERE m.league_id = $1 AND p.user_id = $2
	`, leagueID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPredictions(rows)
}

func (r *predictionRepo) ScoreMatch(ctx context.Context, matchID int64, homeGoals, awayGoals int16) ([]int64, error) {
	// Начисление одним UPDATE; повторный вызов идемпотентен (перезапишет те же значения).
	rows, err := r.db.Query(ctx, `
		UPDATE predictions SET points = CASE
			WHEN home_goals = $2 AND away_goals = $3 THEN 5
			WHEN (home_goals - away_goals) = ($2 - $3) THEN 3
			WHEN SIGN(home_goals - away_goals) = SIGN($2 - $3) THEN 1
			ELSE 0
		END, updated_at = NOW()
		WHERE match_id = $1
		RETURNING user_id, points
	`, matchID, int(homeGoals), int(awayGoals))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var exact []int64
	for rows.Next() {
		var uid int64
		var pts int16
		if err := rows.Scan(&uid, &pts); err != nil {
			return nil, err
		}
		if pts == 5 {
			exact = append(exact, uid)
		}
	}
	return exact, rows.Err()
}

func (r *predictionRepo) Leaderboard(ctx context.Context, leagueID int64) ([]*PredictorRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, u.display_name, COALESCE(u.favorite_club, ''),
		       COALESCE(SUM(p.points), 0) AS pts,
		       COUNT(*) FILTER (WHERE p.points = 5) AS exact,
		       COUNT(*) FILTER (WHERE p.points IS NOT NULL) AS total
		FROM predictions p
		JOIN matches m ON m.id = p.match_id AND m.league_id = $1
		JOIN users u ON u.id = p.user_id
		GROUP BY p.user_id, u.display_name, u.favorite_club
		HAVING COUNT(*) FILTER (WHERE p.points IS NOT NULL) > 0
		ORDER BY pts DESC, exact DESC, total ASC
		LIMIT 50
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*PredictorRow
	for rows.Next() {
		p := &PredictorRow{}
		if err := rows.Scan(&p.UserID, &p.DisplayName, &p.Club, &p.Points, &p.Exact, &p.Total); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *predictionRepo) ByMatch(ctx context.Context, matchID int64) ([]*PredictionRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.match_id, p.user_id, u.display_name, COALESCE(u.favorite_club, ''),
		       p.home_goals, p.away_goals, p.points
		FROM predictions p
		JOIN users u ON u.id = p.user_id
		WHERE p.match_id = $1
		ORDER BY p.points DESC NULLS LAST, p.created_at
	`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPredictions(rows)
}

func (r *predictionRepo) CountForMatch(ctx context.Context, matchID int64) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM predictions WHERE match_id = $1`, matchID).Scan(&n)
	return n, err
}

func (r *predictionRepo) SeasonPoints(ctx context.Context, seasonID int64) (map[int64]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.user_id, COALESCE(SUM(p.points), 0)
		FROM predictions p
		JOIN matches m ON m.id = p.match_id
		JOIN leagues l ON l.id = m.league_id AND l.season_id = $1
		WHERE p.points IS NOT NULL
		GROUP BY p.user_id
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var uid int64
		var pts int
		if err := rows.Scan(&uid, &pts); err != nil {
			return nil, err
		}
		out[uid] = pts
	}
	return out, rows.Err()
}

func collectPredictions(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]*PredictionRow, error) {
	var out []*PredictionRow
	for rows.Next() {
		p := &PredictionRow{}
		if err := rows.Scan(&p.MatchID, &p.UserID, &p.DisplayName, &p.Club,
			&p.HomeGoals, &p.AwayGoals, &p.Points); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
