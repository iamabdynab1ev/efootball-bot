// match_postgres.go - Postgres'да матчларни бошқариш учун репозиторий
package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"
	"fmt"
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

func (r *matchRepo) HasMatches(ctx context.Context, leagueID int64) (bool, error) {
	var count int
	// Считаем только матчи основного этапа (не плей-офф) — защита от повторной жеребьёвки.
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM matches
		WHERE league_id=$1
		  AND stage NOT IN ('sf', 'final', 'qf', 'r16', 'r32', 'r8', '3rd')
	`, leagueID).Scan(&count)
	return count > 0, err
}

// CountUnconfirmedLeagueMatches — сколько матчей основного этапа ещё не подтверждены.
// Работает для всех форматов: лига (stage='league'), группы (stage='A','B'...),
// швейцарская (stage='round_1'...), Лига Наций (stage='div_A'...).
// Плей-офф стадии не считаются.
func (r *matchRepo) CountUnconfirmedLeagueMatches(ctx context.Context, leagueID int64) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM matches
		WHERE league_id = $1
		  AND status != 'confirmed'
		  AND status != 'cancelled'
		  AND stage NOT IN ('sf', 'final', 'qf', 'r16', 'r32', 'r8', '3rd')
	`, leagueID).Scan(&count)
	return count, err
}

func (r *matchRepo) CreateBatch(ctx context.Context, matches []*models.Match) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, m := range matches {
		stage := m.Stage
		if stage == "" {
			stage = "league"
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO matches (league_id, home_user_id, away_user_id, round, stage, bracket_slot)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, m.LeagueID, m.HomeUserID, m.AwayUserID, m.Round, stage, m.BracketSlot)
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
		       COALESCE(uh.telegram_id,0), uh.display_name, COALESCE(uh.favorite_club,'') AS home_club,
		       COALESCE(ua.telegram_id,0), ua.display_name, COALESCE(ua.favorite_club,'') AS away_club
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE m.id = $1
	`, id).Scan(
		&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
		&m.HomeGoals, &m.AwayGoals, &m.ClaimedHome, &m.ClaimedAway,
		&m.Status, &m.DisputeCount, &m.PlayedAt, &m.CreatedAt, &m.UpdatedAt,
		&m.HomeUser.TelegramID, &m.HomeUser.DisplayName, &m.HomeUser.FavoriteClub,
		&m.AwayUser.TelegramID, &m.AwayUser.DisplayName, &m.AwayUser.FavoriteClub,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

// GetAllForLeague — все матчи лиги с именами игроков (для web-расписания).
func (r *matchRepo) GetAllForLeague(ctx context.Context, leagueID int64) ([]*models.Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.league_id, m.home_user_id, m.away_user_id, m.round,
		       m.home_goals, m.away_goals, m.claimed_home, m.claimed_away,
		       m.status, m.dispute_count, m.played_at, m.created_at, m.updated_at,
		       COALESCE(uh.telegram_id,0), uh.display_name, COALESCE(uh.favorite_club,'') AS home_club,
		       COALESCE(ua.telegram_id,0), ua.display_name, COALESCE(ua.favorite_club,'') AS away_club,
		       COALESCE(m.stage,''), m.bracket_slot
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE m.league_id = $1
		ORDER BY m.round ASC, m.id ASC
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Match
	for rows.Next() {
		m := &models.Match{HomeUser: &models.User{}, AwayUser: &models.User{}}
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
			&m.HomeGoals, &m.AwayGoals, &m.ClaimedHome, &m.ClaimedAway,
			&m.Status, &m.DisputeCount, &m.PlayedAt, &m.CreatedAt, &m.UpdatedAt,
			&m.HomeUser.TelegramID, &m.HomeUser.DisplayName, &m.HomeUser.FavoriteClub,
			&m.AwayUser.TelegramID, &m.AwayUser.DisplayName, &m.AwayUser.FavoriteClub,
			&m.Stage, &m.BracketSlot,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
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
		       m.home_goals, m.away_goals, m.claimed_home, m.claimed_away,
		       m.status::text, m.dispute_count, m.played_at, m.created_at, m.updated_at,
		       COALESCE(uh.telegram_id,0), uh.display_name, COALESCE(uh.favorite_club,'') AS home_club,
		       COALESCE(ua.telegram_id,0), ua.display_name, COALESCE(ua.favorite_club,'') AS away_club
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE m.league_id=$1 
		  AND (m.home_user_id=$2 OR m.away_user_id=$2)
		  AND m.status IN ('scheduled', 'disputed', 'pending_confirm')
		ORDER BY m.round ASC
	`, leagueID, userID)
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
			&m.HomeGoals, &m.AwayGoals, &m.ClaimedHome, &m.ClaimedAway,
			&statusStr, &m.DisputeCount, &m.PlayedAt, &m.CreatedAt, &m.UpdatedAt,
			&m.HomeUser.TelegramID, &m.HomeUser.DisplayName, &m.HomeUser.FavoriteClub,
			&m.AwayUser.TelegramID, &m.AwayUser.DisplayName, &m.AwayUser.FavoriteClub,
		); err != nil {
			return nil, err
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

// Confirm — гость подтверждает, финализируем. Возвращает true если строка была обновлена.
func (r *matchRepo) Confirm(ctx context.Context, matchID int64) (bool, error) {
	now := time.Now()
	tag, err := r.db.Exec(ctx, `
		UPDATE matches
		SET home_goals=claimed_home, away_goals=claimed_away,
		    status='confirmed', played_at=$1, updated_at=NOW()
		WHERE id=$2 AND status='pending_confirm'
	`, now, matchID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
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

// AdminResolve — админ решает вручную. Запрещено переразрешать уже подтверждённые матчи.
func (r *matchRepo) AdminResolve(ctx context.Context, matchID int64, homeGoals, awayGoals int16, adminID int64, note string) error {
	now := time.Now()
	tag, err := r.db.Exec(ctx, `
		UPDATE matches
		SET home_goals=$1, away_goals=$2,
		    status='confirmed', dispute_count=0,
		    played_at=$3, updated_at=NOW()
		WHERE id=$4 AND status != 'confirmed'
	`, homeGoals, awayGoals, now, matchID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("match is already confirmed or not found")
	}
	return nil
}

// GetMatchesByStage — все матчи лиги на конкретной стадии (группа "A","B"... или "league").
func (r *matchRepo) GetMatchesByStage(ctx context.Context, leagueID int64, stage string) ([]*models.Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.league_id, m.home_user_id, m.away_user_id, m.round,
		       m.home_goals, m.away_goals, m.claimed_home, m.claimed_away,
		       m.status, m.dispute_count, m.played_at, m.created_at, m.updated_at,
		       COALESCE(uh.telegram_id,0), uh.display_name, COALESCE(uh.favorite_club,'') AS home_club,
		       COALESCE(ua.telegram_id,0), ua.display_name, COALESCE(ua.favorite_club,'') AS away_club
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE m.league_id=$1 AND m.stage=$2
		ORDER BY m.round ASC, m.id ASC
	`, leagueID, stage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Match
	for rows.Next() {
		m := &models.Match{HomeUser: &models.User{}, AwayUser: &models.User{}}
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
			&m.HomeGoals, &m.AwayGoals, &m.ClaimedHome, &m.ClaimedAway,
			&m.Status, &m.DisputeCount, &m.PlayedAt, &m.CreatedAt, &m.UpdatedAt,
			&m.HomeUser.TelegramID, &m.HomeUser.DisplayName, &m.HomeUser.FavoriteClub,
			&m.AwayUser.TelegramID, &m.AwayUser.DisplayName, &m.AwayUser.FavoriteClub,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// GetConfirmedMatchesBetween возвращает подтверждённые матчи между указанными игроками
// внутри одной стадии (группы). Используется для H2H сортировки при равенстве очков.
// stage="" — не фильтровать по стадии (для лиги).
func (r *matchRepo) GetConfirmedMatchesBetween(ctx context.Context, leagueID int64, userIDs []int64, stage string) ([]*models.Match, error) {
	if len(userIDs) < 2 {
		return nil, nil
	}
	// Параметры: $1=leagueID, $2...$N=userIDs, $N+1=stage (если задан)
	args := make([]interface{}, 0, len(userIDs)+2)
	args = append(args, leagueID)
	ph := make([]string, len(userIDs))
	for i, uid := range userIDs {
		args = append(args, uid)
		ph[i] = fmt.Sprintf("$%d", i+2)
	}
	inClause := "(" + joinStrings(ph, ",") + ")"

	stageFilter := ""
	if stage != "" {
		args = append(args, stage)
		stageFilter = fmt.Sprintf("AND stage = $%d", len(args))
	}

	query := fmt.Sprintf(`
		SELECT id, league_id, home_user_id, away_user_id, round,
		       home_goals, away_goals, claimed_home, claimed_away,
		       status, dispute_count, played_at, created_at, updated_at
		FROM matches
		WHERE league_id = $1
		  AND status = 'confirmed'
		  AND home_user_id IN %s
		  AND away_user_id IN %s
		  %s
		ORDER BY id ASC
	`, inClause, inClause, stageFilter)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

func joinStrings(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}

// GetByLeagueStageSlot ищет матч плей-офф по лиге, стадии и номеру слота.
func (r *matchRepo) GetByLeagueStageSlot(ctx context.Context, leagueID int64, stage string, slot int) (*models.Match, error) {
	m := &models.Match{}
	err := r.db.QueryRow(ctx, `
		SELECT id, league_id, home_user_id, away_user_id, round,
		       home_goals, away_goals, claimed_home, claimed_away,
		       status, dispute_count, played_at, created_at, updated_at
		FROM matches
		WHERE league_id=$1 AND stage=$2 AND bracket_slot=$3
		ORDER BY id DESC
		LIMIT 1
	`, leagueID, stage, slot).Scan(
		&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
		&m.HomeGoals, &m.AwayGoals, &m.ClaimedHome, &m.ClaimedAway,
		&m.Status, &m.DisputeCount, &m.PlayedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
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
func (r *matchRepo) GetUserMatchHistory(ctx context.Context, userID int64, limit, offset int, leagueID int64) ([]*models.Match, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	leagueFilter := ""
	args := []any{userID, limit, offset}
	if leagueID > 0 {
		leagueFilter = "AND m.league_id = $4"
		args = append(args, leagueID)
	}
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.league_id, m.home_user_id, m.away_user_id, m.round,
		       m.home_goals, m.away_goals, m.status, m.played_at,
		       uh.display_name, COALESCE(uh.favorite_club,''),
		       ua.display_name, COALESCE(ua.favorite_club,''),
		       COALESCE(m.stage,'')
		FROM matches m
		JOIN users uh ON uh.id = m.home_user_id
		JOIN users ua ON ua.id = m.away_user_id
		WHERE (m.home_user_id = $1 OR m.away_user_id = $1)
		  AND m.status = 'confirmed'
		  `+leagueFilter+`
		ORDER BY m.played_at DESC
		LIMIT $2 OFFSET $3
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.Match
	for rows.Next() {
		m := &models.Match{HomeUser: &models.User{}, AwayUser: &models.User{}}
		homeClub := ""
		awayClub := ""
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.HomeUserID, &m.AwayUserID, &m.Round,
			&m.HomeGoals, &m.AwayGoals, &m.Status, &m.PlayedAt,
			&m.HomeUser.DisplayName, &homeClub,
			&m.AwayUser.DisplayName, &awayClub,
			&m.Stage,
		); err != nil {
			return nil, err
		}
		if homeClub != "" { m.HomeUser.FavoriteClub = &homeClub }
		if awayClub != "" { m.AwayUser.FavoriteClub = &awayClub }
		result = append(result, m)
	}
	return result, rows.Err()
}
// GetAllLeagueForm returns the last 5 confirmed match outcomes (W/D/L) per player in a league.
func (r *matchRepo) GetAllLeagueForm(ctx context.Context, leagueID int64) (map[int64][]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT user_id, home_goals, away_goals, is_home FROM (
			SELECT home_user_id AS user_id, home_goals, away_goals, TRUE AS is_home,
			       ROW_NUMBER() OVER (PARTITION BY home_user_id ORDER BY played_at DESC) AS rn
			FROM matches WHERE league_id=$1 AND status='confirmed' AND home_goals IS NOT NULL
			UNION ALL
			SELECT away_user_id, home_goals, away_goals, FALSE AS is_home,
			       ROW_NUMBER() OVER (PARTITION BY away_user_id ORDER BY played_at DESC) AS rn
			FROM matches WHERE league_id=$1 AND status='confirmed' AND away_goals IS NOT NULL
		) t WHERE rn <= 5 ORDER BY user_id, rn ASC
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int64][]string{}
	for rows.Next() {
		var userID int64
		var homeGoals, awayGoals int16
		var isHome bool
		if err := rows.Scan(&userID, &homeGoals, &awayGoals, &isHome); err != nil {
			return nil, err
		}
		var outcome string
		if isHome {
			if homeGoals > awayGoals {
				outcome = "W"
			} else if homeGoals == awayGoals {
				outcome = "D"
			} else {
				outcome = "L"
			}
		} else {
			if awayGoals > homeGoals {
				outcome = "W"
			} else if awayGoals == homeGoals {
				outcome = "D"
			} else {
				outcome = "L"
			}
		}
		result[userID] = append(result[userID], outcome)
	}
	return result, rows.Err()
}

func (r *matchRepo) GetAllDisputed(ctx context.Context) ([]*models.Match, error) {
	rows, err := r.db.Query(ctx, `
		SELECT m.id, m.league_id, m.home_user_id, m.away_user_id, m.round,
		       m.home_goals, m.away_goals, m.claimed_home, m.claimed_away,
		       m.status::TEXT, m.dispute_count, m.played_at, m.created_at, m.updated_at,
		       COALESCE(uh.telegram_id,0), uh.display_name, COALESCE(uh.favorite_club,'') AS home_club,
		       COALESCE(ua.telegram_id,0), ua.display_name, COALESCE(ua.favorite_club,'') AS away_club
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
			&m.HomeGoals, &m.AwayGoals, &m.ClaimedHome, &m.ClaimedAway,
			&statusStr, &m.DisputeCount, &m.PlayedAt, &m.CreatedAt, &m.UpdatedAt,
			&m.HomeUser.TelegramID, &m.HomeUser.DisplayName, &m.HomeUser.FavoriteClub,
			&m.AwayUser.TelegramID, &m.AwayUser.DisplayName, &m.AwayUser.FavoriteClub,
		); err != nil {
			return nil, err
		}
		m.Status = models.MatchStatus(statusStr)
		result = append(result, m)
	}
	return result, rows.Err()
}
