package repository_test

import (
	"context"
	"os"
	"testing"

	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestAuditRepoE2E проверяет multi-row INSERT с JSONB-метаданными, обратный скан
// (имена через JOIN, round-trip metadata), фильтры и keyset-пагинацию, Prune.
// Требует EFL_TEST_DSN (локальный postgres), иначе пропускается.
func TestAuditRepoE2E(t *testing.T) {
	dsn := os.Getenv("EFL_TEST_DSN")
	if dsn == "" {
		t.Skip("EFL_TEST_DSN не задан — пропускаю интеграционный тест")
	}
	ctx := context.Background()

	testsupport.MigrateLocked(t, dsn, "../../migrations")

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	// Актёр и цель — реальные пользователи (для JOIN имён).
	var actorID, targetID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO users(telegram_id, display_name, created_at) VALUES (980001,'Actor',NOW()) RETURNING id`).
		Scan(&actorID); err != nil {
		t.Fatalf("insert actor: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO users(telegram_id, display_name, created_at) VALUES (980002,'Target',NOW()) RETURNING id`).
		Scan(&targetID); err != nil {
		t.Fatalf("insert target: %v", err)
	}

	repo := repository.NewAuditRepository(pool)

	entries := []*models.AuditEntry{
		{ActorID: &actorID, Action: models.AuditLogin, Metadata: map[string]any{"method": "google"}, IP: "1.2.3.4"},
		{ActorID: &actorID, Action: models.AuditAdminResolve, EntityType: "match", TargetID: &targetID,
			Metadata: map[string]any{"home_goals": 2, "away_goals": 1, "note": "разбор спора"}},
	}
	if err := repo.InsertBatch(ctx, entries); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}
	for i, e := range entries {
		if e.ID == 0 {
			t.Fatalf("entry %d: id не проставлен после вставки", i)
		}
		if e.CreatedAt.IsZero() {
			t.Fatalf("entry %d: created_at не проставлен", i)
		}
	}

	// Список по актору: обе записи, имена через JOIN, metadata round-trip.
	got, err := repo.List(ctx, models.AuditFilter{ActorID: &actorID})
	if err != nil {
		t.Fatalf("List by actor: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ожидал 2 записи, получил %d", len(got))
	}
	// Порядок id DESC: первой идёт последняя вставленная (admin_resolve).
	first := got[0]
	if first.ActorName != "Actor" {
		t.Fatalf("actor name не подтянулось: %q", first.ActorName)
	}
	if first.Action != models.AuditAdminResolve || first.TargetName != "Target" {
		t.Fatalf("неожиданная запись: action=%s target=%s", first.Action, first.TargetName)
	}
	if first.Metadata["note"] != "разбор спора" {
		t.Fatalf("metadata round-trip сломан: %#v", first.Metadata)
	}

	// Фильтр по action.
	byAction, err := repo.List(ctx, models.AuditFilter{Action: models.AuditLogin})
	if err != nil || len(byAction) != 1 {
		t.Fatalf("фильтр по action: n=%d err=%v", len(byAction), err)
	}
	if byAction[0].IP != "1.2.3.4" {
		t.Fatalf("ip не сохранился: %q", byAction[0].IP)
	}

	// Keyset-пагинация: before = id первой → должна вернуть только более старую.
	page, err := repo.List(ctx, models.AuditFilter{ActorID: &actorID, BeforeID: first.ID, Limit: 10})
	if err != nil || len(page) != 1 {
		t.Fatalf("keyset before: n=%d err=%v", len(page), err)
	}
	if page[0].ID >= first.ID {
		t.Fatalf("keyset вернул не более старую запись: %d >= %d", page[0].ID, first.ID)
	}

	// Prune с огромным сроком ничего не удаляет (записи свежие).
	if n, err := repo.Prune(ctx, 3650); err != nil || n != 0 {
		t.Fatalf("Prune(3650): n=%d err=%v (ожидал 0)", n, err)
	}
}
