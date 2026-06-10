package repository

import (
	"context"
	"efootball-bot/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type awardRepo struct {
	db *pgxpool.Pool
}

func NewAwardRepository(db *pgxpool.Pool) AwardRepository {
	return &awardRepo{db: db}
}

func (r *awardRepo) CreateAward(ctx context.Context, seasonID, leagueID int64, awardType string, userID int64, value int) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO season_awards (season_id, league_id, award_type, user_id, value)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (league_id, award_type) WHERE league_id IS NOT NULL
		DO UPDATE SET season_id = EXCLUDED.season_id, user_id = EXCLUDED.user_id,
		              value = EXCLUDED.value, created_at = NOW()
	`, seasonID, leagueID, awardType, userID, value)
	return err
}

func (r *awardRepo) GetBySeason(ctx context.Context, seasonID int64) ([]*models.SeasonAward, error) {
	return r.query(ctx, `
		SELECT sa.id, sa.season_id, sa.league_id, sa.award_type, sa.user_id, sa.value, sa.created_at,
		       u.display_name, COALESCE(l.name,''), COALESCE(s.name,'')
		FROM season_awards sa
		JOIN users u ON u.id = sa.user_id
		LEFT JOIN leagues l ON l.id = sa.league_id
		JOIN seasons s ON s.id = sa.season_id
		WHERE sa.season_id = $1
		ORDER BY sa.created_at DESC
	`, seasonID)
}

func (r *awardRepo) GetAll(ctx context.Context) ([]*models.SeasonAward, error) {
	return r.query(ctx, `
		SELECT sa.id, sa.season_id, sa.league_id, sa.award_type, sa.user_id, sa.value, sa.created_at,
		       u.display_name, COALESCE(l.name,''), COALESCE(s.name,'')
		FROM season_awards sa
		JOIN users u ON u.id = sa.user_id
		LEFT JOIN leagues l ON l.id = sa.league_id
		JOIN seasons s ON s.id = sa.season_id
		ORDER BY sa.season_id DESC, sa.created_at DESC
	`)
}

func (r *awardRepo) query(ctx context.Context, sql string, args ...any) ([]*models.SeasonAward, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.SeasonAward
	for rows.Next() {
		a := &models.SeasonAward{}
		if err := rows.Scan(
			&a.ID, &a.SeasonID, &a.LeagueID, &a.AwardType, &a.UserID, &a.Value, &a.CreatedAt,
			&a.UserDisplayName, &a.LeagueName, &a.SeasonName,
		); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
