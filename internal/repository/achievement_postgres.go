package repository

import (
	"context"
	"efootball-bot/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type achievementRepo struct {
	db *pgxpool.Pool
}

func NewAchievementRepository(db *pgxpool.Pool) AchievementRepository {
	return &achievementRepo{db: db}
}

func (r *achievementRepo) GetAll(ctx context.Context) ([]*models.Achievement, error) {
	rows, err := r.db.Query(ctx, `SELECT id, code, icon, name_uz, name_ru, name_tg FROM achievements ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.Achievement
	for rows.Next() {
		a := &models.Achievement{}
		if err := rows.Scan(&a.ID, &a.Code, &a.Icon, &a.NameUz, &a.NameRu, &a.NameTg); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *achievementRepo) GetUserAchievements(ctx context.Context, userID int64) ([]*models.UserAchievement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ua.id, ua.user_id, ua.achievement_id, ua.league_id, ua.earned_at,
		       a.code, a.icon, a.name_uz, a.name_ru, a.name_tg
		FROM user_achievements ua
		JOIN achievements a ON a.id = ua.achievement_id
		WHERE ua.user_id = $1
		ORDER BY ua.earned_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.UserAchievement
	for rows.Next() {
		ua := &models.UserAchievement{Achievement: &models.Achievement{}}
		if err := rows.Scan(
			&ua.ID, &ua.UserID, &ua.AchievementID, &ua.LeagueID, &ua.EarnedAt,
			&ua.Achievement.Code, &ua.Achievement.Icon,
			&ua.Achievement.NameUz, &ua.Achievement.NameRu, &ua.Achievement.NameTg,
		); err != nil {
			return nil, err
		}
		result = append(result, ua)
	}
	return result, rows.Err()
}

func (r *achievementRepo) HasAchievement(ctx context.Context, userID int64, code string, leagueID *int64) (bool, error) {
	var count int
	var err error
	if leagueID == nil {
		err = r.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM user_achievements ua
			JOIN achievements a ON a.id = ua.achievement_id
			WHERE ua.user_id=$1 AND a.code=$2 AND ua.league_id IS NULL
		`, userID, code).Scan(&count)
	} else {
		err = r.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM user_achievements ua
			JOIN achievements a ON a.id = ua.achievement_id
			WHERE ua.user_id=$1 AND a.code=$2 AND ua.league_id=$3
		`, userID, code, *leagueID).Scan(&count)
	}
	return count > 0, err
}

func (r *achievementRepo) Award(ctx context.Context, userID int64, code string, leagueID *int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_achievements (user_id, achievement_id, league_id)
		SELECT $1, a.id, $3 FROM achievements a WHERE a.code=$2
		ON CONFLICT DO NOTHING
	`, userID, code, leagueID)
	return err
}
