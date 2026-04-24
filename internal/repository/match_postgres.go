// match_postgres.go - Postgres'да матчларни бошқариш учун репозиторий
package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type matchRepo struct {
	db *pgxpool.Pool
}

func NewMatchRepository(db *pgxpool.Pool) MatchRepository {
	return &matchRepo{db: db}
}

func (r *matchRepo) CreateBatch(ctx context.Context, matches []*models.Match) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, m := range matches {
		_, err := tx.Exec(ctx, `
			INSERT INTO matches (league_id, home_user_id, away_user_id, round)
			VALUES ($1, $2, $3, $4)
		`, m.LeagueID, m.HomeUserID, m.AwayUserID, m.Round)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *matchRepo) GetByID(ctx context.Context, id int64) (*models.Match, error) {
	m := &models.Match{HomeUser: &models.User{}, AwayUser: &models.User{}}
	err := r.db.QueryRow(ctx, `
		SELECT m.id, m.league_id, m.home_user_id, m.away_user_id, m.round,
		       m.home_goals, m.away_goals, m.claimed_home, m.claimed_away,
		       m.status, m.dispute_count, m.played_at, m.created_at, m.updated_at,
		       uh.telegram_id, uh.display_name,
		       ua.telegram_id, ua.display_name
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE m.id = $1
	`, id).Scan(
		&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
		&m.HomeGoals, &m.AwayGoals, &m.ClaimedHome, &m.ClaimedAway,
		&m.Status, &m.DisputeCount, &m.PlayedAt, &m.CreatedAt, &m.UpdatedAt,
		&m.HomeUser.TelegramID, &m.HomeUser.DisplayName,
		&m.AwayUser.TelegramID, &m.AwayUser.DisplayName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

// GetPendingForUser — матчи где игрок должен подтвердить или ввести счёт
func (r *matchRepo) GetPendingForUser(ctx context.Context, userID int64) ([]*models.Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, league_id, home_user_id, away_user_id, round,
		       home_goals, away_goals, claimed_home, claimed_away,
		       status, dispute_count, played_at, created_at, updated_at
		FROM matches
		WHERE (home_user_id=$1 OR away_user_id=$1)
		  AND status IN ('pending_confirm','disputed')
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatches(rows)
}

func (r *matchRepo) GetScheduleForLeague(ctx context.Context, leagueID int64, round int16) ([]*models.Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, league_id, home_user_id, away_user_id, round,
		       home_goals, away_goals, claimed_home, claimed_away,
		       status, dispute_count, played_at, created_at, updated_at
		FROM matches
		WHERE league_id=$1 AND round=$2
		ORDER BY id ASC
	`, leagueID, round)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatches(rows)
}

func (r *matchRepo) GetUserSchedule(ctx context.Context, userID, leagueID int64) ([]*models.Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.league_id, m.home_user_id, m.away_user_id, m.round,
		       m.status::text, uh.display_name, ua.display_name
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE m.league_id=$1 
		  AND (m.home_user_id=$2 OR m.away_user_id=$2)
		  AND m.status::text IN ('scheduled', 'disputed', 'pending_confirm') 
		ORDER BY m.round ASC
	`, leagueID, userID)
	if err != nil {
		log.Printf("SQL Error in GetUserSchedule: %v", err)
		return nil, err
	}
	defer rows.Close()

	var result []*models.Match
	for rows.Next() {
		m := &models.Match{HomeUser: &models.User{}, AwayUser: &models.User{}}
		var statusStr string
		err := rows.Scan(
			&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
			&statusStr, &m.HomeUser.DisplayName, &m.AwayUser.DisplayName,
		)
		if err != nil {
			log.Printf("Scan Error in GetUserSchedule: %v", err)
			continue
		}
		m.Status = models.MatchStatus(statusStr)
		result = append(result, m)
	}
	return result, rows.Err()
}

// ClaimResult — хозяин вводит счёт
func (r *matchRepo) ClaimResult(ctx context.Context, matchID int64, homeGoals, awayGoals int16) error {
	_, err := r.db.Exec(ctx, `
		UPDATE matches
		SET claimed_home=$1, claimed_away=$2,
		    status='pending_confirm', updated_at=NOW()
		WHERE id=$3 AND status IN ('scheduled','disputed')
	`, homeGoals, awayGoals, matchID)
	return err
}

// Confirm — гость подтверждает, финализируем
func (r *matchRepo) Confirm(ctx context.Context, matchID int64) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE matches
		SET home_goals=claimed_home, away_goals=claimed_away,
		    status='confirmed', played_at=$1, updated_at=NOW()
		WHERE id=$2 AND status='pending_confirm'
	`, now, matchID)
	return err
}

// Dispute — гость не согласен, возвращаем хозяину
func (r *matchRepo) Dispute(ctx context.Context, matchID int64, homeClaimed, awayClaimed int16) error {
	_, err := r.db.Exec(ctx, `
		UPDATE matches
		SET status='disputed',
		    dispute_count=dispute_count+1,
		    updated_at=NOW()
		WHERE id=$1 AND status='pending_confirm'
	`, matchID)
	return err
}

// AdminResolve — админ решает вручную
func (r *matchRepo) AdminResolve(ctx context.Context, matchID int64, homeGoals, awayGoals int16, adminID int64, note string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE matches
		SET home_goals=$1, away_goals=$2,
		    status='confirmed', played_at=$3, updated_at=NOW()
		WHERE id=$4
	`, homeGoals, awayGoals, now, matchID)
	return err
}

func scanMatches(rows pgx.Rows) ([]*models.Match, error) {
	var result []*models.Match
	for rows.Next() {
		m := &models.Match{}
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
			&m.HomeGoals, &m.AwayGoals, &m.ClaimedHome, &m.ClaimedAway,
			&m.Status, &m.DisputeCount, &m.PlayedAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
func (r *matchRepo) GetUserMatchHistory(ctx context.Context, userID int64) ([]*models.Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.league_id, m.home_user_id, m.away_user_id, m.round,
		       m.home_goals, m.away_goals, m.status, m.played_at,
		       uh.display_name, ua.display_name
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE (m.home_user_id = $1 OR m.away_user_id = $1)
		  AND m.status = 'confirmed'
		ORDER BY m.played_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Match
	for rows.Next() {
		m := &models.Match{HomeUser: &models.User{}, AwayUser: &models.User{}}
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
			&m.HomeGoals, &m.AwayGoals, &m.Status, &m.PlayedAt,
			&m.HomeUser.DisplayName, &m.AwayUser.DisplayName,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
func (r *matchRepo) GetAllDisputed(ctx context.Context) ([]*models.Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.league_id, m.home_user_id, m.away_user_id, m.round,
		       m.status::text,
		       uh.display_name, ua.display_name
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE m.status = 'disputed'
		ORDER BY m.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Match
	for rows.Next() {
		m := &models.Match{HomeUser: &models.User{}, AwayUser: &models.User{}}
		var statusStr string
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
			&statusStr, &m.HomeUser.DisplayName, &m.AwayUser.DisplayName,
		); err != nil {
			log.Printf("Scan Error in GetAllDisputed: %v", err)
			continue
		}
		m.Status = models.MatchStatus(statusStr)
		result = append(result, m)
	}
	return result, rows.Err()
}
