// league_postgres.go - Postgres'да лигалар ва уларнинг аъзоларини бошқариш учун репозиторий
package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type leagueRepo struct {
	db *pgxpool.Pool
}

func NewLeagueRepository(db *pgxpool.Pool) LeagueRepository {
	return &leagueRepo{db: db}
}
func (r *leagueRepo) GetAllPendingMembers(ctx context.Context) ([]*models.LeagueMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.id, lm.league_id, lm.user_id, lm.status::text,
		       u.display_name, l.name as league_name
		FROM league_members lm
		JOIN users u ON u.id = lm.user_id
		JOIN leagues l ON l.id = lm.league_id
		WHERE lm.status = 'pending'
		  AND l.status != 'archived'
		ORDER BY lm.joined_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{
			User:   &models.User{},
			League: &models.League{},
		}
		var statusStr string
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &statusStr,
			&m.User.DisplayName, &m.League.Name,
		); err != nil {
			return nil, err
		}
		m.Status = models.MemberStatus(statusStr)
		result = append(result, m)
	}
	return result, nil
}

func (r *leagueRepo) GetActiveLeagues(ctx context.Context) ([]*models.League, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, season_id, name, country, level, max_players, rounds_type, status::text, registration_deadline, created_at, updated_at
		FROM leagues
		WHERE status != 'archived'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.League
	for rows.Next() {
		l := &models.League{}
		err := rows.Scan(&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level, &l.MaxPlayers, &l.RoundsType, &l.Status, &l.RegistrationDeadline, &l.CreatedAt, &l.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, nil
}

// 2. Лигани архив қилиш (статусини ўзгартириш)
func (r *leagueRepo) ArchiveLeague(ctx context.Context, leagueID int64) error {
	_, err := r.db.Exec(ctx, `UPDATE leagues SET status = 'archived' WHERE id = $1`, leagueID)
	return err
}

// 3. Лигани бутунлай ўчириш
func (r *leagueRepo) DeleteLeague(ctx context.Context, leagueID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM leagues WHERE id = $1`, leagueID)
	return err
}
func (r *leagueRepo) GetByID(ctx context.Context, id int64) (*models.League, error) {
	l := &models.League{}
	err := r.db.QueryRow(ctx, `
		SELECT id, season_id, name, country, level, max_players, rounds_type,
		       COALESCE(num_groups,0), COALESCE(group_advance,1), COALESCE(best_runners_up,0),
		       COALESCE(best_of,1),
		       status, registration_deadline, created_at, updated_at
		FROM leagues WHERE id=$1
	`, id).Scan(&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level,
		&l.MaxPlayers, &l.RoundsType, &l.NumGroups, &l.GroupAdvance, &l.BestRunnersUp,
		&l.BestOf,
		&l.Status, &l.RegistrationDeadline, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return l, err
}

func (r *leagueRepo) GetByName(ctx context.Context, name string) (*models.League, error) {
	l := &models.League{}
	err := r.db.QueryRow(ctx, `
		SELECT id, season_id, name, country, level, max_players, rounds_type, status, registration_deadline, created_at, updated_at
		FROM leagues WHERE LOWER(name)=LOWER($1)
	`, name).Scan(&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level,
		&l.MaxPlayers, &l.RoundsType, &l.Status, &l.RegistrationDeadline, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return l, err
}

func (r *leagueRepo) AddMember(ctx context.Context, leagueID, userID int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO league_members (league_id, user_id, status)
		VALUES ($1, $2, 'pending')
		ON CONFLICT (league_id, user_id) DO NOTHING
	`, leagueID, userID)
	return err
}

func (r *leagueRepo) ApproveMember(ctx context.Context, leagueID, userID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// FOR UPDATE блокирует строку лиги — предотвращает одновременный одобрение сверх лимита.
	var maxPlayers, currentCount int
	err = tx.QueryRow(ctx, `
		SELECT l.max_players,
		       (SELECT COUNT(*) FROM league_members WHERE league_id=$1 AND status='approved')
		FROM leagues l WHERE l.id=$1
		FOR UPDATE
	`, leagueID).Scan(&maxPlayers, &currentCount)
	if err != nil {
		return err
	}
	if currentCount >= maxPlayers {
		return errors.New("league is full")
	}

	_, err = tx.Exec(ctx, `
		UPDATE league_members SET status='approved', updated_at=NOW()
		WHERE league_id=$1 AND user_id=$2
	`, leagueID, userID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *leagueRepo) RejectMember(ctx context.Context, leagueID, userID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE league_members SET status='rejected', updated_at=NOW()
		WHERE league_id=$1 AND user_id=$2
	`, leagueID, userID)
	return err
}

func (r *leagueRepo) GetPendingMembers(ctx context.Context, leagueID int64) ([]*models.LeagueMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.id, lm.league_id, lm.user_id, lm.status::text,
		       lm.points, lm.wins, lm.draws, lm.losses,
		       lm.goals_for, lm.goals_against, lm.position,
		       lm.joined_at, lm.updated_at,
		       u.id, COALESCE(u.telegram_id,0), u.display_name, u.username,
		       u.rating, u.team_power, u.rank
		FROM league_members lm
		JOIN users u ON u.id = lm.user_id
		WHERE lm.league_id=$1 AND lm.status='pending'
		ORDER BY lm.joined_at ASC
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{User: &models.User{}}
		var statusStr string
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &statusStr,
			&m.Points, &m.Wins, &m.Draws, &m.Losses,
			&m.GoalsFor, &m.GoalsAgainst, &m.Position,
			&m.JoinedAt, &m.UpdatedAt,
			&m.User.ID, &m.User.TelegramID, &m.User.DisplayName, &m.User.Username,
			&m.User.Rating, &m.User.TeamPower, &m.User.Rank,
		); err != nil {
			return nil, err
		}
		m.Status = models.MemberStatus(statusStr)
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *leagueRepo) GetMembers(ctx context.Context, leagueID int64) ([]*models.LeagueMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.id, lm.league_id, lm.user_id, lm.status::text,
		       lm.points, lm.wins, lm.draws, lm.losses,
		       lm.goals_for, lm.goals_against, lm.position,
		       lm.joined_at, lm.updated_at,
		       u.id, COALESCE(u.telegram_id,0), u.display_name, u.username,
		       u.rating, u.team_power, u.rank, u.favorite_club,
		       COALESCE(lm.group_name,'') AS group_name
		FROM league_members lm
		JOIN users u ON u.id = lm.user_id
		WHERE lm.league_id = $1 AND lm.status = 'approved'
		ORDER BY lm.points DESC, (lm.goals_for - lm.goals_against) DESC, lm.goals_for DESC
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{User: &models.User{}}
		var statusStr string
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &statusStr,
			&m.Points, &m.Wins, &m.Draws, &m.Losses,
			&m.GoalsFor, &m.GoalsAgainst, &m.Position,
			&m.JoinedAt, &m.UpdatedAt,
			&m.User.ID, &m.User.TelegramID, &m.User.DisplayName, &m.User.Username,
			&m.User.Rating, &m.User.TeamPower, &m.User.Rank, &m.User.FavoriteClub,
			&m.GroupName,
		); err != nil {
			return nil, err
		}
		m.Status = models.MemberStatus(statusStr)
		result = append(result, m)
	}
	return result, nil
}
func (r *leagueRepo) ApplyMatchResultStats(ctx context.Context, leagueID, homeUserID, awayUserID int64, hg, ag int16) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var hW, hD, hL, hP int16
	var aW, aD, aL, aP int16

	if hg > ag {
		hW, hP, aL = 1, 3, 1 // Хозяин победил
	} else if hg < ag {
		aW, aP, hL = 1, 3, 1 // Гость победил
	} else {
		hD, hP, aD, aP = 1, 1, 1, 1 // Ничья
	}

	// Атомарное обновление для Хозяина (Никогда не сбрасывает очки!)
	_, err = tx.Exec(ctx, `
		UPDATE league_members 
		SET wins = wins + $1, draws = draws + $2, losses = losses + $3, points = points + $4,
		    goals_for = goals_for + $5, goals_against = goals_against + $6, updated_at = NOW()
		WHERE league_id = $7 AND user_id = $8`,
		hW, hD, hL, hP, hg, ag, leagueID, homeUserID)
	if err != nil {
		return err
	}

	// Атомарное обновление для Гостя
	_, err = tx.Exec(ctx, `
		UPDATE league_members 
		SET wins = wins + $1, draws = draws + $2, losses = losses + $3, points = points + $4,
		    goals_for = goals_for + $5, goals_against = goals_against + $6, updated_at = NOW()
		WHERE league_id = $7 AND user_id = $8`,
		aW, aD, aL, aP, ag, hg, leagueID, awayUserID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *leagueRepo) IsMember(ctx context.Context, leagueID, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM league_members
			WHERE league_id=$1 AND user_id=$2 AND status != 'rejected'
		)
	`, leagueID, userID).Scan(&exists)
	return exists, err
}

func (r *leagueRepo) GetMemberStats(ctx context.Context, leagueID, userID int64) (*models.LeagueMember, error) {
	m := &models.LeagueMember{}
	err := r.db.QueryRow(ctx, `
		SELECT id, league_id, user_id, status,
		       points, wins, draws, losses,
		       goals_for, goals_against, position,
		       joined_at, updated_at
		FROM league_members
		WHERE league_id=$1 AND user_id=$2
	`, leagueID, userID).Scan(
		&m.ID, &m.LeagueID, &m.UserID, &m.Status,
		&m.Points, &m.Wins, &m.Draws, &m.Losses,
		&m.GoalsFor, &m.GoalsAgainst, &m.Position,
		&m.JoinedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return m, err
}

// RecalculateTable — пересчитывает позиции в таблице после матча
func (r *leagueRepo) RecalculateTable(ctx context.Context, leagueID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE league_members lm
		SET position = sub.pos, updated_at = NOW()
		FROM (
			SELECT id,
			       ROW_NUMBER() OVER (
			           ORDER BY points DESC,
			                    (goals_for - goals_against) DESC,
			                    goals_for DESC
			       ) AS pos
			FROM league_members
			WHERE league_id = $1 AND status = 'approved'
		) sub
		WHERE lm.id = sub.id
	`, leagueID)
	return err
}

func (r *leagueRepo) SetMemberPosition(ctx context.Context, memberID int64, position int16) error {
	_, err := r.db.Exec(ctx,
		`UPDATE league_members SET position=$1, updated_at=NOW() WHERE id=$2`,
		position, memberID,
	)
	return err
}

func scanMembers(rows pgx.Rows) ([]*models.LeagueMember, error) {
	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{}
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &m.Status,
			&m.Points, &m.Wins, &m.Draws, &m.Losses,
			&m.GoalsFor, &m.GoalsAgainst, &m.Position,
			&m.JoinedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *leagueRepo) GetOrCreateActiveSeason(ctx context.Context) (*models.Season, error) {
	s := &models.Season{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, status, created_at, updated_at
		FROM seasons WHERE status='active' LIMIT 1
	`).Scan(&s.ID, &s.Name, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err == nil {
		return s, nil
	}
	// Создаём новый сезон
	err = r.db.QueryRow(ctx, `
		INSERT INTO seasons (name, status) VALUES ('Сезон 1', 'active')
		RETURNING id, name, status, created_at, updated_at
	`).Scan(&s.ID, &s.Name, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

func (r *leagueRepo) CreateLeague(ctx context.Context, seasonID int64, name string, deadline *time.Time, roundsType string, numGroups, groupAdvance, bestRunnersUp int16) (*models.League, error) {
	if groupAdvance <= 0 {
		groupAdvance = 1
	}
	// Если дедлайн указан — сразу открываем набор, иначе черновик
	initialStatus := "draft"
	if deadline != nil {
		initialStatus = "registration"
	}
	l := &models.League{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO leagues (season_id, name, status, rounds_type, level, max_players,
		                     registration_deadline, num_groups, group_advance, best_runners_up)
		VALUES ($1, $2, $3, $4, 1, 20, $5, $6, $7, $8)
		RETURNING id, season_id, name, country, level, max_players, rounds_type,
		          COALESCE(num_groups,0), COALESCE(group_advance,1), COALESCE(best_runners_up,0),
		          status, registration_deadline, created_at, updated_at
	`, seasonID, name, initialStatus, roundsType, deadline, numGroups, groupAdvance, bestRunnersUp).Scan(
		&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level,
		&l.MaxPlayers, &l.RoundsType, &l.NumGroups, &l.GroupAdvance, &l.BestRunnersUp,
		&l.Status, &l.RegistrationDeadline, &l.CreatedAt, &l.UpdatedAt,
	)
	return l, err
}

func (r *leagueRepo) UpdateLeague(ctx context.Context, id int64, name string, deadline *time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE leagues
		SET name=$1, registration_deadline=$2, updated_at=NOW()
		WHERE id=$3
	`, name, deadline, id)
	return err
}

func (r *leagueRepo) SetLeagueStatus(ctx context.Context, leagueID int64, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE leagues SET status=$1, updated_at=NOW() WHERE id=$2
	`, status, leagueID)
	return err
}

func (r *leagueRepo) GetAllLeagues(ctx context.Context) ([]*models.League, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, season_id, name, country, level, max_players, rounds_type, status, registration_deadline, created_at, updated_at
		FROM leagues ORDER BY status, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.League
	for rows.Next() {
		l := &models.League{}
		if err := rows.Scan(&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level,
			&l.MaxPlayers, &l.RoundsType, &l.Status, &l.RegistrationDeadline, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, rows.Err()
}
func (r *leagueRepo) RemoveMember(ctx context.Context, leagueID, userID int64) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM league_members 
		WHERE league_id = $1 AND user_id = $2 
		AND league_id IN (SELECT id FROM leagues WHERE status = 'registration')
	`, leagueID, userID)
	return err
}
func (r *leagueRepo) GetUserLeagues(ctx context.Context, userID int64) ([]*models.LeagueMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.id, lm.league_id, lm.user_id, lm.status::text,
		       lm.points, lm.wins, lm.draws, lm.losses,
		       lm.goals_for, lm.goals_against, lm.position,
		       lm.joined_at, lm.updated_at,
		       l.id, l.name, l.status::text
		FROM league_members lm
		JOIN leagues l ON l.id = lm.league_id
		WHERE lm.user_id = $1
		  AND lm.status IN ('approved', 'pending')
		  AND l.status NOT IN ('archived', 'finished')
		ORDER BY lm.joined_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{}
		var statusStr, leagueStatusStr string
		league := &models.League{}

		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &statusStr,
			&m.Points, &m.Wins, &m.Draws, &m.Losses,
			&m.GoalsFor, &m.GoalsAgainst, &m.Position,
			&m.JoinedAt, &m.UpdatedAt,
			&league.ID, &league.Name, &leagueStatusStr,
		); err != nil {
			return nil, err
		}
		m.Status = models.MemberStatus(statusStr)
		league.Status = models.LeagueStatus(leagueStatusStr)
		// Временно храним лигу в отдельном поле через User (чисто)
		// Используем отдельный трюк — возвращаем League через LeagueID
		// Сохраняем лигу прямо в структуру через доп поле
		m.League = league
		result = append(result, m)
	}
	return result, rows.Err()
}

// ── Group Stage ───────────────────────────────────────────────────────────────

func (r *leagueRepo) SetMemberGroups(ctx context.Context, leagueID int64, userIDs []int64, groups []string) error {
	if len(userIDs) == 0 || len(userIDs) != len(groups) {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE league_members lm
		SET group_name = v.group_name, updated_at = NOW()
		FROM (SELECT unnest($2::bigint[]) AS user_id, unnest($3::text[]) AS group_name) v
		WHERE lm.league_id = $1 AND lm.user_id = v.user_id
	`, leagueID, userIDs, groups)
	return err
}

func (r *leagueRepo) GetMembersByGroup(ctx context.Context, leagueID int64, groupName string) ([]*models.LeagueMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.id, lm.league_id, lm.user_id, lm.status::text,
		       lm.points, lm.wins, lm.draws, lm.losses,
		       lm.goals_for, lm.goals_against, lm.position,
		       lm.joined_at, lm.updated_at,
		       u.id, COALESCE(u.telegram_id,0), u.display_name, u.username,
		       u.rating, u.team_power, u.rank, u.favorite_club,
		       COALESCE(lm.group_name,'') AS group_name
		FROM league_members lm
		JOIN users u ON u.id = lm.user_id
		WHERE lm.league_id=$1 AND lm.group_name=$2 AND lm.status='approved'
		ORDER BY lm.points DESC, (lm.goals_for-lm.goals_against) DESC, lm.goals_for DESC
	`, leagueID, groupName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{User: &models.User{}}
		var statusStr, gName string
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &statusStr,
			&m.Points, &m.Wins, &m.Draws, &m.Losses,
			&m.GoalsFor, &m.GoalsAgainst, &m.Position,
			&m.JoinedAt, &m.UpdatedAt,
			&m.User.ID, &m.User.TelegramID, &m.User.DisplayName, &m.User.Username,
			&m.User.Rating, &m.User.TeamPower, &m.User.Rank, &m.User.FavoriteClub,
			&gName,
		); err != nil {
			return nil, err
		}
		m.Status = models.MemberStatus(statusStr)
		m.GroupName = gName
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *leagueRepo) SetCurrentRound(ctx context.Context, leagueID int64, round int16) error {
	_, err := r.db.Exec(ctx, `UPDATE leagues SET current_round=$1, updated_at=NOW() WHERE id=$2`, round, leagueID)
	return err
}

func (r *leagueRepo) SetLeagueGroupConfig(ctx context.Context, leagueID int64, numGroups, groupAdvance int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE leagues SET num_groups=$1, group_advance=$2, updated_at=NOW() WHERE id=$3`,
		numGroups, groupAdvance, leagueID)
	return err
}

func (r *leagueRepo) SetLeagueBestOf(ctx context.Context, leagueID int64, bestOf int) error {
	if bestOf < 1 {
		bestOf = 1
	}
	_, err := r.db.Exec(ctx,
		`UPDATE leagues SET best_of=$1, updated_at=NOW() WHERE id=$2`, bestOf, leagueID)
	return err
}

func (r *leagueRepo) GetLeagueGroups(ctx context.Context, leagueID int64) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT group_name FROM league_members
		WHERE league_id=$1 AND group_name IS NOT NULL AND group_name != ''
		ORDER BY group_name
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// ── Nations League ────────────────────────────────────────────────────────────

func (r *leagueRepo) SetMemberDivision(ctx context.Context, leagueID, userID int64, division string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE league_members SET division_name=$1, updated_at=NOW()
		WHERE league_id=$2 AND user_id=$3
	`, division, leagueID, userID)
	return err
}

func (r *leagueRepo) GetMembersByDivision(ctx context.Context, leagueID int64, division string) ([]*models.LeagueMember, error) {
	rows, err := r.db.Query(ctx, `
		SELECT lm.id, lm.league_id, lm.user_id, lm.status::text,
		       lm.points, lm.wins, lm.draws, lm.losses,
		       lm.goals_for, lm.goals_against, lm.position,
		       lm.joined_at, lm.updated_at,
		       u.id, COALESCE(u.telegram_id,0), u.display_name, u.username,
		       u.rating, u.team_power, u.rank, u.favorite_club,
		       COALESCE(lm.division_name,'') AS division_name
		FROM league_members lm
		JOIN users u ON u.id = lm.user_id
		WHERE lm.league_id=$1 AND lm.division_name=$2 AND lm.status='approved'
		ORDER BY lm.points DESC, (lm.goals_for-lm.goals_against) DESC, lm.goals_for DESC
	`, leagueID, division)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{User: &models.User{}}
		var statusStr, divName string
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &statusStr,
			&m.Points, &m.Wins, &m.Draws, &m.Losses,
			&m.GoalsFor, &m.GoalsAgainst, &m.Position,
			&m.JoinedAt, &m.UpdatedAt,
			&m.User.ID, &m.User.TelegramID, &m.User.DisplayName, &m.User.Username,
			&m.User.Rating, &m.User.TeamPower, &m.User.Rank, &m.User.FavoriteClub,
			&divName,
		); err != nil {
			return nil, err
		}
		m.Status = models.MemberStatus(statusStr)
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *leagueRepo) GetLeagueDivisions(ctx context.Context, leagueID int64) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT division_name FROM league_members
		WHERE league_id=$1 AND division_name IS NOT NULL AND division_name != ''
		ORDER BY division_name
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// GetGroupRunnersUp возвращает игроков, занявших место (groupAdvance+1) в каждой
// группе — то есть первых "за бортом" прямого прохода. Используется для выбора
// "лучших вторых мест" (wildcard) в формате FIFA. groupAdvance должен совпадать
// с числом мест прямого прохода, иначе можно получить дубликаты с advancingIDs.
func (r *leagueRepo) GetGroupRunnersUp(ctx context.Context, leagueID int64, groupAdvance int) ([]*models.LeagueMember, error) {
	if groupAdvance < 1 {
		groupAdvance = 1
	}
	rows, err := r.db.Query(ctx, `
		WITH group_ranked AS (
			SELECT lm.*, u.display_name, u.rating, u.team_power, u.rank, u.favorite_club,
			       COALESCE(u.telegram_id,0) AS tg_id, u.username,
			       ROW_NUMBER() OVER (
			           PARTITION BY lm.group_name
			           ORDER BY lm.points DESC,
			                    (lm.goals_for - lm.goals_against) DESC,
			                    lm.goals_for DESC
			       ) AS pos
			FROM league_members lm
			JOIN users u ON u.id = lm.user_id
			WHERE lm.league_id = $1 AND lm.status = 'approved'
			  AND lm.group_name IS NOT NULL AND lm.group_name != ''
		)
		SELECT id, league_id, user_id, status::text,
		       points, wins, draws, losses,
		       goals_for, goals_against, position,
		       joined_at, updated_at,
		       u_id, tg_id, display_name, username,
		       rating, team_power, rank, favorite_club,
		       group_name
		FROM group_ranked
		CROSS JOIN LATERAL (SELECT id AS u_id FROM users WHERE id = user_id) u
		WHERE pos = $2
		ORDER BY points DESC, (goals_for - goals_against) DESC, goals_for DESC
	`, leagueID, groupAdvance+1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{User: &models.User{}}
		var statusStr, gName string
		if err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &statusStr,
			&m.Points, &m.Wins, &m.Draws, &m.Losses,
			&m.GoalsFor, &m.GoalsAgainst, &m.Position,
			&m.JoinedAt, &m.UpdatedAt,
			&m.User.ID, &m.User.TelegramID, &m.User.DisplayName, &m.User.Username,
			&m.User.Rating, &m.User.TeamPower, &m.User.Rank, &m.User.FavoriteClub,
			&gName,
		); err != nil {
			return nil, err
		}
		m.Status = models.MemberStatus(statusStr)
		m.GroupName = gName
		result = append(result, m)
	}
	return result, rows.Err()
}
