package repository

import (
	"context"
	"efootball-bot/internal/models"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type deRepo struct {
	db *pgxpool.Pool
}

func NewDoubleElimRepository(db *pgxpool.Pool) DoubleElimRepository {
	return &deRepo{db: db}
}

func (r *deRepo) HasDoubleElim(ctx context.Context, leagueID int64) (bool, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM de_nodes WHERE league_id=$1`, leagueID).Scan(&n)
	return n > 0, err
}

// createDEMatch вставляет матч стадии double-elim и возвращает его id.
func createDEMatch(ctx context.Context, tx pgx.Tx, leagueID, home, away int64, stage string, round int) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO matches (league_id, home_user_id, away_user_id, round, stage)
		VALUES ($1,$2,$3,$4,$5) RETURNING id
	`, leagueID, home, away, int16(round), stage).Scan(&id)
	return id, err
}

func (r *deRepo) GenerateDoubleElim(ctx context.Context, leagueID int64, nodes []*models.DENode) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, bracketLockClass, int32(leagueID)); err != nil {
		return err
	}

	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM de_nodes WHERE league_id=$1`, leagueID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrBracketExists
	}

	for _, n := range nodes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO de_nodes
			  (league_id, node_key, bracket, round, ord, is_reset, home_user_id, away_user_id, home_src, away_src)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		`, leagueID, n.NodeKey, n.Bracket, n.Round, n.Ord, n.IsReset,
			n.HomeUserID, n.AwayUserID, n.HomeSrc, n.AwaySrc); err != nil {
			return err
		}
	}

	// Стартовые матчи: узлы, у которых оба участника уже известны (WB-раунд 1),
	// кроме reset.
	for _, n := range nodes {
		if n.IsReset || n.HomeUserID == nil || n.AwayUserID == nil {
			continue
		}
		mid, err := createDEMatch(ctx, tx, leagueID, *n.HomeUserID, *n.AwayUserID, n.Bracket, n.Round)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE de_nodes SET match_id=$1 WHERE league_id=$2 AND node_key=$3`,
			mid, leagueID, n.NodeKey); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// deNodeRow — минимальный набор полей узла для логики продвижения.
type deNodeRow struct {
	id         int64
	nodeKey    int
	bracket    string
	round      int
	isReset    bool
	homeUserID *int64
	awayUserID *int64
	homeSrc    *string
	awaySrc    *string
	matchID    *int64
	winnerID   *int64
}

func (r *deRepo) AdvanceDoubleElim(ctx context.Context, leagueID, matchID, winnerID, loserID int64) (*int64, []*models.Match, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, bracketLockClass, int32(leagueID)); err != nil {
		return nil, nil, err
	}

	// Узел этого матча.
	var node deNodeRow
	err = tx.QueryRow(ctx, `
		SELECT id, node_key, bracket, round, is_reset, home_user_id, away_user_id, match_id, winner_user_id
		FROM de_nodes WHERE league_id=$1 AND match_id=$2
	`, leagueID, matchID).Scan(&node.id, &node.nodeKey, &node.bracket, &node.round, &node.isReset,
		&node.homeUserID, &node.awayUserID, &node.matchID, &node.winnerID)
	if err == pgx.ErrNoRows {
		return nil, nil, tx.Commit(ctx) // не DE-матч
	}
	if err != nil {
		return nil, nil, err
	}
	if node.winnerID != nil {
		return nil, nil, tx.Commit(ctx) // уже продвинут (идемпотентность)
	}

	if _, err := tx.Exec(ctx, `UPDATE de_nodes SET winner_user_id=$1 WHERE id=$2`, winnerID, node.id); err != nil {
		return nil, nil, err
	}

	// ── Гранд-финал ──────────────────────────────────────────────────────────
	if node.bracket == models.StageDEGrand {
		if node.isReset {
			// Решающий матч сыгран — победитель чемпион.
			return &winnerID, nil, tx.Commit(ctx)
		}
		// GF1: home — представитель верхней сетки.
		if node.homeUserID != nil && winnerID == *node.homeUserID {
			// Верхняя сетка победила без поражений — турнир окончен.
			return &winnerID, nil, tx.Commit(ctx)
		}
		// Победил представитель нижней сетки → bracket reset.
		var resetID int64
		var resetKey, resetRound int
		err := tx.QueryRow(ctx, `
			SELECT id, node_key, round FROM de_nodes
			WHERE league_id=$1 AND is_reset=TRUE
		`, leagueID).Scan(&resetID, &resetKey, &resetRound)
		if err == pgx.ErrNoRows {
			return &winnerID, nil, tx.Commit(ctx) // reset-узла нет — считаем чемпионом
		}
		if err != nil {
			return nil, nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE de_nodes SET home_user_id=$1, away_user_id=$2 WHERE id=$3`,
			winnerID, loserID, resetID); err != nil {
			return nil, nil, err
		}
		mid, err := createDEMatch(ctx, tx, leagueID, winnerID, loserID, models.StageDEGrand, resetRound)
		if err != nil {
			return nil, nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE de_nodes SET match_id=$1 WHERE id=$2`, mid, resetID); err != nil {
			return nil, nil, err
		}
		created := &models.Match{ID: mid, LeagueID: leagueID, HomeUserID: winnerID, AwayUserID: loserID, Stage: models.StageDEGrand, Round: int16(resetRound)}
		return nil, []*models.Match{created}, tx.Commit(ctx)
	}

	// ── Обычные узлы (WB/LB): маршрутизация победителя и проигравшего ─────────
	winKey := fmt.Sprintf("win:%d", node.nodeKey)
	loseKey := fmt.Sprintf("lose:%d", node.nodeKey)

	rows, err := tx.Query(ctx, `
		SELECT id, node_key, bracket, round, is_reset, home_user_id, away_user_id, home_src, away_src, match_id
		FROM de_nodes
		WHERE league_id=$1 AND (home_src=$2 OR home_src=$3 OR away_src=$2 OR away_src=$3)
	`, leagueID, winKey, loseKey)
	if err != nil {
		return nil, nil, err
	}
	var consumers []deNodeRow
	for rows.Next() {
		var c deNodeRow
		if err := rows.Scan(&c.id, &c.nodeKey, &c.bracket, &c.round, &c.isReset,
			&c.homeUserID, &c.awayUserID, &c.homeSrc, &c.awaySrc, &c.matchID); err != nil {
			rows.Close()
			return nil, nil, err
		}
		consumers = append(consumers, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var created []*models.Match
	for _, c := range consumers {
		if c.isReset {
			continue // reset активируется только через GF1
		}
		newHome := c.homeUserID
		newAway := c.awayUserID
		if c.homeSrc != nil && *c.homeSrc == winKey {
			newHome = &winnerID
		} else if c.homeSrc != nil && *c.homeSrc == loseKey {
			newHome = &loserID
		}
		if c.awaySrc != nil && *c.awaySrc == winKey {
			newAway = &winnerID
		} else if c.awaySrc != nil && *c.awaySrc == loseKey {
			newAway = &loserID
		}
		if _, err := tx.Exec(ctx, `UPDATE de_nodes SET home_user_id=$1, away_user_id=$2 WHERE id=$3`,
			newHome, newAway, c.id); err != nil {
			return nil, nil, err
		}
		// Оба участника известны и матча ещё нет → создаём.
		if newHome != nil && newAway != nil && c.matchID == nil {
			mid, err := createDEMatch(ctx, tx, leagueID, *newHome, *newAway, c.bracket, c.round)
			if err != nil {
				return nil, nil, err
			}
			if _, err := tx.Exec(ctx, `UPDATE de_nodes SET match_id=$1 WHERE id=$2`, mid, c.id); err != nil {
				return nil, nil, err
			}
			created = append(created, &models.Match{
				ID: mid, LeagueID: leagueID, HomeUserID: *newHome, AwayUserID: *newAway,
				Stage: c.bracket, Round: int16(c.round),
			})
		}
	}

	return nil, created, tx.Commit(ctx)
}

func (r *deRepo) GetDoubleElimNodes(ctx context.Context, leagueID int64) ([]*models.DENode, error) {
	rows, err := r.db.Query(ctx, `
		SELECT dn.id, dn.league_id, dn.node_key, dn.bracket, dn.round, dn.ord, dn.is_reset,
		       dn.home_user_id, dn.away_user_id, dn.home_src, dn.away_src, dn.match_id, dn.winner_user_id,
		       COALESCE(uh.display_name,''), COALESCE(ua.display_name,''), COALESCE(uw.display_name,''),
		       COALESCE(uh.favorite_club,''), COALESCE(ua.favorite_club,''),
		       m.home_goals, m.away_goals, COALESCE(m.status::text,'')
		FROM de_nodes dn
		LEFT JOIN users uh ON uh.id = dn.home_user_id
		LEFT JOIN users ua ON ua.id = dn.away_user_id
		LEFT JOIN users uw ON uw.id = dn.winner_user_id
		LEFT JOIN matches m ON m.id = dn.match_id
		WHERE dn.league_id=$1
		ORDER BY
		    CASE dn.bracket WHEN 'de_w' THEN 1 WHEN 'de_l' THEN 2 WHEN 'de_gf' THEN 3 END,
		    dn.round, dn.ord
	`, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*models.DENode
	for rows.Next() {
		n := &models.DENode{}
		if err := rows.Scan(&n.ID, &n.LeagueID, &n.NodeKey, &n.Bracket, &n.Round, &n.Ord, &n.IsReset,
			&n.HomeUserID, &n.AwayUserID, &n.HomeSrc, &n.AwaySrc, &n.MatchID, &n.WinnerUserID,
			&n.HomeName, &n.AwayName, &n.WinnerName,
			&n.HomeClub, &n.AwayClub,
			&n.HomeGoals, &n.AwayGoals, &n.MatchStatus); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, rows.Err()
}
