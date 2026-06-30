package repository

import (
	"context"
	"efootball-bot/internal/models"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository interface {
	// InsertBatch пишет пачку записей одним multi-row INSERT и проставляет
	// сгенерированные id обратно в entries (для последующей живой публикации).
	InsertBatch(ctx context.Context, entries []*models.AuditEntry) error
	// List возвращает ленту по фильтру (keyset-пагинация по id DESC) с JOIN
	// имён актора/цели для отображения.
	List(ctx context.Context, f models.AuditFilter) ([]*models.AuditEntry, error)
	// Prune удаляет записи старше keepDays — контроль роста под лимит Neon 0.5GB.
	Prune(ctx context.Context, keepDays int) (int64, error)
}

type auditRepo struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) AuditRepository {
	return &auditRepo{db: db}
}

func (r *auditRepo) InsertBatch(ctx context.Context, entries []*models.AuditEntry) error {
	if len(entries) == 0 {
		return nil
	}
	// Один multi-row INSERT ... RETURNING id вместо N запросов — держит число
	// обращений к БД постоянным при всплеске активности.
	var b strings.Builder
	b.WriteString(`INSERT INTO audit_log (actor_id, action, entity_type, entity_id, league_id, target_id, metadata, ip) VALUES `)
	args := make([]any, 0, len(entries)*8)
	for i, e := range entries {
		if i > 0 {
			b.WriteByte(',')
		}
		n := i * 8
		b.WriteString("($" + strconv.Itoa(n+1) + ",$" + strconv.Itoa(n+2) + ",$" + strconv.Itoa(n+3) +
			",$" + strconv.Itoa(n+4) + ",$" + strconv.Itoa(n+5) + ",$" + strconv.Itoa(n+6) +
			",$" + strconv.Itoa(n+7) + ",$" + strconv.Itoa(n+8) + ")")
		args = append(args, e.ActorID, e.Action, nullStr(e.EntityType), e.EntityID,
			e.LeagueID, e.TargetID, metadataArg(e.Metadata), nullStr(e.IP))
	}
	b.WriteString(" RETURNING id, created_at")

	rows, err := r.db.Query(ctx, b.String(), args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		if err := rows.Scan(&entries[i].ID, &entries[i].CreatedAt); err != nil {
			return err
		}
		i++
	}
	return rows.Err()
}

func (r *auditRepo) List(ctx context.Context, f models.AuditFilter) ([]*models.AuditEntry, error) {
	var b strings.Builder
	b.WriteString(`
		SELECT a.id, a.actor_id, actor.display_name, a.action, a.entity_type,
		       a.entity_id, a.league_id, a.target_id, target.display_name,
		       a.metadata, a.ip, a.created_at
		FROM audit_log a
		LEFT JOIN users actor  ON actor.id  = a.actor_id
		LEFT JOIN users target ON target.id = a.target_id
		WHERE 1=1`)
	args := make([]any, 0, 6)
	add := func(cond string, v any) {
		args = append(args, v)
		b.WriteString(" AND " + cond + "$" + strconv.Itoa(len(args)))
	}
	if f.ActorID != nil {
		add("a.actor_id=", *f.ActorID)
	}
	if f.TargetID != nil {
		add("a.target_id=", *f.TargetID)
	}
	if f.LeagueID != nil {
		add("a.league_id=", *f.LeagueID)
	}
	if f.Action != "" {
		add("a.action=", f.Action)
	}
	if f.BeforeID > 0 {
		add("a.id<", f.BeforeID)
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit)
	b.WriteString(" ORDER BY a.id DESC LIMIT $" + strconv.Itoa(len(args)))

	rows, err := r.db.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.AuditEntry
	for rows.Next() {
		e := &models.AuditEntry{}
		var actorName, targetName, entityType, ip *string
		var meta []byte
		if err := rows.Scan(&e.ID, &e.ActorID, &actorName, &e.Action, &entityType,
			&e.EntityID, &e.LeagueID, &e.TargetID, &targetName,
			&meta, &ip, &e.CreatedAt); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &e.Metadata)
		}
		if actorName != nil {
			e.ActorName = *actorName
		}
		if targetName != nil {
			e.TargetName = *targetName
		}
		if entityType != nil {
			e.EntityType = *entityType
		}
		if ip != nil {
			e.IP = *ip
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *auditRepo) Prune(ctx context.Context, keepDays int) (int64, error) {
	if keepDays <= 0 {
		return 0, nil
	}
	tag, err := r.db.Exec(ctx,
		`DELETE FROM audit_log WHERE created_at < NOW() - make_interval(days => $1)`,
		keepDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// nullStr превращает пустую строку в NULL — не засоряем колонки пустотой.
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// metadataArg маршалит map в JSON-байты для JSONB-колонки; nil-map → NULL.
func metadataArg(m map[string]any) any {
	if len(m) == 0 {
		return nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return b
}
