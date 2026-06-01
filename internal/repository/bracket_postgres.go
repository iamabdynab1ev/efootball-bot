package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type bracketRepo struct {
	db *pgxpool.Pool
}

func NewBracketRepository(db *pgxpool.Pool) BracketRepository {
	return &bracketRepo{db: db}
}

func (r *bracketRepo) HasBracket(ctx context.Context, leagueID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM bracket_slots WHERE league_id=$1`, leagueID).Scan(&count)
	return count > 0, err
}

func (r *bracketRepo) CreateSlots(ctx context.Context, slots []*models.BracketSlot) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, s := range slots {
		_, err := tx.Exec(ctx, `
			INSERT INTO bracket_slots (league_id, stage, slot, home_user_id, away_user_id)
			VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (league_id, stage, slot) DO NOTHING
		`, s.LeagueID, s.Stage, s.Slot, s.HomeUserID, s.AwayUserID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *bracketRepo) GetSlot(ctx context.Context, leagueID int64, stage string, slot int) (*models.BracketSlot, error) {
	row := &models.BracketSlot{}
	err := r.db.QueryRow(ctx, `
		SELECT bs.id, bs.league_id, bs.stage, bs.slot,
		       bs.home_user_id, bs.away_user_id, bs.match_id, bs.winner_user_id,
		       COALESCE(uh.display_name,''), COALESCE(ua.display_name,''), COALESCE(uw.display_name,''),
		       m.home_goals, m.away_goals, COALESCE(m.status::text,'')
		FROM bracket_slots bs
		LEFT JOIN users uh ON uh.id = bs.home_user_id
		LEFT JOIN users ua ON ua.id = bs.away_user_id
		LEFT JOIN users uw ON uw.id = bs.winner_user_id
		LEFT JOIN matches m ON m.id = bs.match_id
		WHERE bs.league_id=$1 AND bs.stage=$2 AND bs.slot=$3
	`, leagueID, stage, slot).Scan(
		&row.ID, &row.LeagueID, &row.Stage, &row.Slot,
		&row.HomeUserID, &row.AwayUserID, &row.MatchID, &row.WinnerUserID,
		&row.HomeName, &row.AwayName, &row.WinnerName,
		&row.HomeGoals, &row.AwayGoals, &row.MatchStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return row, err
}

func (r *bracketRepo) GetAllSlots(ctx context.Context, leagueID int64) ([]*models.BracketSlot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT bs.id, bs.league_id, bs.stage, bs.slot,
		       bs.home_user_id, bs.away_user_id, bs.match_id, bs.winner_user_id,
		       COALESCE(uh.display_name,''), COALESCE(ua.display_name,''), COALESCE(uw.display_name,''),
		       m.home_goals, m.away_goals, COALESCE(m.status::text,'')
		FROM bracket_slots bs
		LEFT JOIN users uh ON uh.id = bs.home_user_id
		LEFT JOIN users ua ON ua.id = bs.away_user_id
		LEFT JOIN users uw ON uw.id = bs.winner_user_id
		LEFT JOIN matches m ON m.id = bs.match_id
		WHERE bs.league_id=$1
		ORDER BY
		    CASE bs.stage
		        WHEN 'r32'   THEN 1
		        WHEN 'r16'   THEN 2
		        WHEN 'qf'    THEN 3
		        WHEN 'sf'    THEN 4
		        WHEN 'final' THEN 5
		    END, bs.slot
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.BracketSlot
	for rows.Next() {
		row := &models.BracketSlot{}
		if err := rows.Scan(
			&row.ID, &row.LeagueID, &row.Stage, &row.Slot,
			&row.HomeUserID, &row.AwayUserID, &row.MatchID, &row.WinnerUserID,
			&row.HomeName, &row.AwayName, &row.WinnerName,
			&row.HomeGoals, &row.AwayGoals, &row.MatchStatus,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *bracketRepo) SetWinner(ctx context.Context, leagueID int64, stage string, slot int, winnerID, matchID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE bracket_slots SET winner_user_id=$1, match_id=$2
		WHERE league_id=$3 AND stage=$4 AND slot=$5
	`, winnerID, matchID, leagueID, stage, slot)
	return err
}

func (r *bracketRepo) SetParticipant(ctx context.Context, leagueID int64, stage string, slot int, userID int64, isHome bool) error {
	if isHome {
		_, err := r.db.Exec(ctx, `
			UPDATE bracket_slots SET home_user_id=$1
			WHERE league_id=$2 AND stage=$3 AND slot=$4
		`, userID, leagueID, stage, slot)
		return err
	}
	_, err := r.db.Exec(ctx, `
		UPDATE bracket_slots SET away_user_id=$1
		WHERE league_id=$2 AND stage=$3 AND slot=$4
	`, userID, leagueID, stage, slot)
	return err
}

func (r *bracketRepo) SetMatchID(ctx context.Context, leagueID int64, stage string, slot int, matchID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE bracket_slots SET match_id=$1
		WHERE league_id=$2 AND stage=$3 AND slot=$4
	`, matchID, leagueID, stage, slot)
	return err
}
