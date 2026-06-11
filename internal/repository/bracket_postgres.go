package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrBracketExists возвращается GenerateBracket, если у лиги уже есть сетка.
// Проверка выполняется внутри транзакции под advisory-lock, поэтому две
// параллельные генерации не могут пройти обе.
var ErrBracketExists = errors.New("playoff bracket already generated")

// bracketLockClass — пространство имён advisory-lock для записей в сетку:
// генерация и продвижение одной лиги сериализуются между собой.
const bracketLockClass = 7001

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

func (r *bracketRepo) GenerateBracket(ctx context.Context, leagueID int64, slots []*models.BracketSlot, matches []*models.Match) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, bracketLockClass, int32(leagueID)); err != nil {
		return err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM bracket_slots WHERE league_id=$1`, leagueID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrBracketExists
	}

	for _, s := range slots {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bracket_slots (league_id, stage, slot, home_user_id, away_user_id)
			VALUES ($1,$2,$3,$4,$5)
		`, s.LeagueID, s.Stage, s.Slot, s.HomeUserID, s.AwayUserID); err != nil {
			return err
		}
	}

	for _, m := range matches {
		stage := m.Stage
		if stage == "" {
			stage = "league"
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO matches (league_id, home_user_id, away_user_id, round, stage, bracket_slot)
			VALUES ($1,$2,$3,$4,$5,$6)
			RETURNING id
		`, m.LeagueID, m.HomeUserID, m.AwayUserID, m.Round, stage, m.BracketSlot).Scan(&m.ID); err != nil {
			return err
		}
		if m.BracketSlot != nil {
			if _, err := tx.Exec(ctx, `
				UPDATE bracket_slots SET match_id=$1
				WHERE league_id=$2 AND stage=$3 AND slot=$4
			`, m.ID, m.LeagueID, stage, *m.BracketSlot); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (r *bracketRepo) AdvanceSlot(ctx context.Context, p AdvanceParams) (*models.Match, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, bracketLockClass, int32(p.LeagueID)); err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE bracket_slots SET winner_user_id=$1, match_id=$2
		WHERE league_id=$3 AND stage=$4 AND slot=$5
	`, p.WinnerID, p.MatchID, p.LeagueID, p.Stage, p.Slot); err != nil {
		return nil, err
	}

	// Финал сыгран — продвигать некуда.
	if p.NextStage == "" {
		return nil, tx.Commit(ctx)
	}

	side := "away_user_id"
	if p.IsHome {
		side = "home_user_id"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bracket_slots SET `+side+`=$1
		WHERE league_id=$2 AND stage=$3 AND slot=$4
	`, p.WinnerID, p.LeagueID, p.NextStage, p.NextSlot); err != nil {
		return nil, err
	}

	var homeID, awayID *int64
	var matchID *int64
	err = tx.QueryRow(ctx, `
		SELECT home_user_id, away_user_id, match_id FROM bracket_slots
		WHERE league_id=$1 AND stage=$2 AND slot=$3
	`, p.LeagueID, p.NextStage, p.NextSlot).Scan(&homeID, &awayID, &matchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tx.Commit(ctx)
	}
	if err != nil {
		return nil, err
	}
	if homeID == nil || awayID == nil || matchID != nil {
		// Слот не готов или матч уже создан.
		return nil, tx.Commit(ctx)
	}

	created := &models.Match{
		LeagueID:    p.LeagueID,
		HomeUserID:  *homeID,
		AwayUserID:  *awayID,
		Round:       p.NewRound,
		Stage:       p.NextStage,
		BracketSlot: &p.NextSlot,
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO matches (league_id, home_user_id, away_user_id, round, stage, bracket_slot)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id
	`, created.LeagueID, created.HomeUserID, created.AwayUserID, created.Round, created.Stage, created.BracketSlot).Scan(&created.ID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bracket_slots SET match_id=$1
		WHERE league_id=$2 AND stage=$3 AND slot=$4
	`, created.ID, p.LeagueID, p.NextStage, p.NextSlot); err != nil {
		return nil, err
	}
	return created, tx.Commit(ctx)
}

func (r *bracketRepo) GetAllSlots(ctx context.Context, leagueID int64) ([]*models.BracketSlot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT bs.id, bs.league_id, bs.stage, bs.slot,
		       bs.home_user_id, bs.away_user_id, bs.match_id, bs.winner_user_id,
		       COALESCE(uh.display_name,''), COALESCE(ua.display_name,''), COALESCE(uw.display_name,''),
		       COALESCE(uh.favorite_club,''), COALESCE(ua.favorite_club,''), COALESCE(uw.favorite_club,''),
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
			&row.HomeClub, &row.AwayClub, &row.WinnerClub,
			&row.HomeGoals, &row.AwayGoals, &row.MatchStatus,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
