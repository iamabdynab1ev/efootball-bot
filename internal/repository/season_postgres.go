package repository

import (
	"context"
	"time"

	"efootball-bot/internal/models"

	"github.com/jackc/pgx/v5"
)

// Сезонные запросы: агрегаты игроков по всем лигам сезона, закрытие сезона
// с созданием следующего, рост ELO за период. Живут на leagueRepo — он уже
// владеет таблицей seasons.

// SeasonAggregate — суммарные показатели игрока по всем лигам сезона.
type SeasonAggregate struct {
	UserID       int64
	DisplayName  string
	FavoriteClub string
	Points       int
	Wins         int
	Draws        int
	Losses       int
	GoalsFor     int
	GoalsAgainst int
	Leagues      int
}

func (r *leagueRepo) GetSeasonByID(ctx context.Context, id int64) (*models.Season, error) {
	s := &models.Season{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, status, created_at, updated_at, closed_at
		FROM seasons WHERE id=$1
	`, id).Scan(&s.ID, &s.Name, &s.Status, &s.CreatedAt, &s.UpdatedAt, &s.ClosedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// GetLatestClosedSeason — последний закрытый сезон (для баннера церемонии).
func (r *leagueRepo) GetLatestClosedSeason(ctx context.Context) (*models.Season, error) {
	s := &models.Season{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, status, created_at, updated_at, closed_at
		FROM seasons WHERE status='finished' AND closed_at IS NOT NULL
		ORDER BY closed_at DESC LIMIT 1
	`).Scan(&s.ID, &s.Name, &s.Status, &s.CreatedAt, &s.UpdatedAt, &s.ClosedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *leagueRepo) ListLeaguesBySeason(ctx context.Context, seasonID int64) ([]*models.League, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, status FROM leagues WHERE season_id=$1 ORDER BY id
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.League
	for rows.Next() {
		l := &models.League{SeasonID: seasonID}
		if err := rows.Scan(&l.ID, &l.Name, &l.Status); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// SeasonAggregates — суммы очков/голов игроков по всем лигам сезона.
func (r *leagueRepo) SeasonAggregates(ctx context.Context, seasonID int64) ([]*SeasonAggregate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.user_id, u.display_name, COALESCE(u.favorite_club,''),
		       SUM(lm.points), SUM(lm.wins), SUM(lm.draws), SUM(lm.losses),
		       SUM(lm.goals_for), SUM(lm.goals_against), COUNT(DISTINCT lm.league_id)
		FROM league_members lm
		JOIN leagues l ON l.id = lm.league_id AND l.season_id = $1
		JOIN users u ON u.id = lm.user_id
		WHERE lm.status = 'approved'
		GROUP BY lm.user_id, u.display_name, u.favorite_club
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*SeasonAggregate
	for rows.Next() {
		a := &SeasonAggregate{}
		if err := rows.Scan(&a.UserID, &a.DisplayName, &a.FavoriteClub,
			&a.Points, &a.Wins, &a.Draws, &a.Losses,
			&a.GoalsFor, &a.GoalsAgainst, &a.Leagues); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// SeasonTotals — «за сезон: N матчей, M голов» для заставки церемонии.
func (r *leagueRepo) SeasonTotals(ctx context.Context, seasonID int64) (matches, goals int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(m.home_goals + m.away_goals), 0)
		FROM matches m
		JOIN leagues l ON l.id = m.league_id AND l.season_id = $1
		WHERE m.status = 'confirmed'
	`, seasonID).Scan(&matches, &goals)
	return matches, goals, err
}

// EloDeltasSince — рост рейтинга каждого игрока с момента начала сезона
// (последняя запись истории минус первая в окне).
func (r *leagueRepo) EloDeltasSince(ctx context.Context, since time.Time) (map[int64]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id,
		       (ARRAY_AGG(rating ORDER BY id DESC))[1] - (ARRAY_AGG(rating ORDER BY id ASC))[1]
		FROM rating_history
		WHERE created_at >= $1
		GROUP BY user_id
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var uid int64
		var delta int
		if err := rows.Scan(&uid, &delta); err != nil {
			return nil, err
		}
		out[uid] = delta
	}
	return out, rows.Err()
}

// CloseSeason закрывает сезон и открывает следующий одной транзакцией.
func (r *leagueRepo) CloseSeason(ctx context.Context, seasonID int64, nextName string) (*models.Season, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE seasons SET status='finished', closed_at=NOW(), updated_at=NOW()
		WHERE id=$1 AND status='active'
	`, seasonID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}

	next := &models.Season{}
	if err := tx.QueryRow(ctx, `
		INSERT INTO seasons (name, status) VALUES ($1, 'active')
		RETURNING id, name, status, created_at, updated_at, closed_at
	`, nextName).Scan(&next.ID, &next.Name, &next.Status, &next.CreatedAt, &next.UpdatedAt, &next.ClosedAt); err != nil {
		return nil, err
	}
	return next, tx.Commit(ctx)
}
