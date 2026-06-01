// league_postgres.go - Postgres'да лигалар ва уларнинг аъзоларини бошқариш учун репозиторий
package repository

import (
	"context"
	"efootball-bot/internal/models"
	"errors"
	"log"

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
		WHERE lm.status::text = 'pending'
		  AND l.status::text != 'archived'
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
			continue
		}
		m.Status = models.MemberStatus(statusStr)
		result = append(result, m)
	}
	return result, nil
}

func (r *leagueRepo) GetActiveLeagues(ctx context.Context) ([]*models.League, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, season_id, name, country, level, max_players, rounds_type, status::text, created_at, updated_at
		FROM leagues 
		WHERE status::text != 'archived'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*models.League
	for rows.Next() {
		l := &models.League{}
		err := rows.Scan(&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level, &l.MaxPlayers, &l.RoundsType, &l.Status, &l.CreatedAt, &l.UpdatedAt)
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
		SELECT id, season_id, name, country, level, max_players, rounds_type, status, created_at, updated_at
		FROM leagues WHERE id=$1
	`, id).Scan(&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level,
		&l.MaxPlayers, &l.RoundsType, &l.Status, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return l, err
}

func (r *leagueRepo) GetByName(ctx context.Context, name string) (*models.League, error) {
	l := &models.League{}
	err := r.db.QueryRow(ctx, `
		SELECT id, season_id, name, country, level, max_players, rounds_type, status, created_at, updated_at
		FROM leagues WHERE LOWER(name)=LOWER($1)
	`, name).Scan(&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level,
		&l.MaxPlayers, &l.RoundsType, &l.Status, &l.CreatedAt, &l.UpdatedAt)
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

	var maxPlayers, currentCount int
	err = tx.QueryRow(ctx, `
		SELECT l.max_players,
		       (SELECT COUNT(*) FROM league_members WHERE league_id=$1 AND status='approved')
		FROM leagues l WHERE l.id=$1
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
		       u.rating, u.team_power, u.rank
		FROM league_members lm
		JOIN users u ON u.id = lm.user_id
		WHERE lm.league_id = $1 AND lm.status::text = 'approved'
		ORDER BY lm.points DESC, (lm.goals_for - lm.goals_against) DESC, lm.goals_for DESC
	`, leagueID)
	if err != nil {
		log.Printf("SQL Error in GetMembers: %v", err)
		return nil, err
	}
	defer rows.Close()

	var result []*models.LeagueMember
	for rows.Next() {
		m := &models.LeagueMember{User: &models.User{}}
		// lm.statusни string қилиб оламиз ва Null бўлиши мумкин бўлган устунларга эътибор берамиз
		var statusStr string
		err := rows.Scan(
			&m.ID, &m.LeagueID, &m.UserID, &statusStr,
			&m.Points, &m.Wins, &m.Draws, &m.Losses,
			&m.GoalsFor, &m.GoalsAgainst, &m.Position,
			&m.JoinedAt, &m.UpdatedAt,
			&m.User.ID, &m.User.TelegramID, &m.User.DisplayName, &m.User.Username,
			&m.User.Rating, &m.User.TeamPower, &m.User.Rank,
		)
		if err != nil {
			log.Printf("Scan Error in GetMembers: %v", err)
			continue // Битта қаторда хато бўлса, кейингисига ўтамиз
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
			WHERE league_id=$1 AND user_id=$2 AND status::text != 'rejected'
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

func (r *leagueRepo) CreateLeague(ctx context.Context, seasonID int64, name string) (*models.League, error) {
	l := &models.League{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO leagues (season_id, name, status, rounds_type, level, max_players)
		VALUES ($1, $2, 'registration', 'double', 1, 20)
		RETURNING id, season_id, name, country, level, max_players, rounds_type, status, created_at, updated_at
	`, seasonID, name).Scan(
		&l.ID, &l.SeasonID, &l.Name, &l.Country, &l.Level,
		&l.MaxPlayers, &l.RoundsType, &l.Status, &l.CreatedAt, &l.UpdatedAt,
	)
	return l, err
}

func (r *leagueRepo) SetLeagueStatus(ctx context.Context, leagueID int64, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE leagues SET status=$1, updated_at=NOW() WHERE id=$2
	`, status, leagueID)
	return err
}

func (r *leagueRepo) GetAllLeagues(ctx context.Context) ([]*models.League, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, season_id, name, country, level, max_players, rounds_type, status, created_at, updated_at
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
			&l.MaxPlayers, &l.RoundsType, &l.Status, &l.CreatedAt, &l.UpdatedAt); err != nil {
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
		AND league_id IN (SELECT id FROM leagues WHERE status::text = 'registration')
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
		  AND lm.status = 'approved'
		  AND l.status != 'archived'
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
			log.Printf("Scan Error in GetUserLeagues: %v", err)
			continue
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
