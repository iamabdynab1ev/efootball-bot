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

// TestNotificationRepoE2E проверяет batch-вставку с RETURNING, ленту,
// счётчик непрочитанных, точечный и массовый mark-read, prune прочитанных.
func TestNotificationRepoE2E(t *testing.T) {
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

	var uid int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO users(telegram_id, display_name, created_at) VALUES (970777,'NotifUser',NOW()) RETURNING id`).
		Scan(&uid); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	repo := repository.NewNotificationRepository(pool)

	items := []*models.Notification{
		{UserID: uid, Type: models.NotifMatchResult, Title: "Результат", Body: "2:1", Link: "/leagues/details?id=1"},
		{UserID: uid, Type: models.NotifMatchConfirmed, Title: "Подтверждён", Body: "2:1"},
	}
	if err := repo.CreateBatch(ctx, items); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	for i, n := range items {
		if n.ID == 0 || n.CreatedAt.IsZero() || n.Read {
			t.Fatalf("item %d: некорректные дефолты после вставки (%+v)", i, n)
		}
	}

	// Лента: обе записи, id DESC, поля сохранены.
	list, err := repo.ListByUser(ctx, uid, 0, 30)
	if err != nil || len(list) != 2 {
		t.Fatalf("ListByUser: n=%d err=%v", len(list), err)
	}
	if list[0].ID <= list[1].ID {
		t.Fatal("лента не отсортирована по id DESC")
	}
	if list[1].Link != "/leagues/details?id=1" {
		t.Fatalf("link не сохранился: %q", list[1].Link)
	}

	// Непрочитанных — 2.
	if n, err := repo.CountUnread(ctx, uid); err != nil || n != 2 {
		t.Fatalf("CountUnread: n=%d err=%v (ожидал 2)", n, err)
	}

	// Точечный mark-read одной записи → остаётся 1 непрочитанная.
	if err := repo.MarkRead(ctx, uid, []int64{items[0].ID}); err != nil {
		t.Fatalf("MarkRead one: %v", err)
	}
	if n, _ := repo.CountUnread(ctx, uid); n != 1 {
		t.Fatalf("после точечного mark-read ожидал 1, получил %d", n)
	}

	// Массовый mark-read (ids=nil) → 0 непрочитанных.
	if err := repo.MarkRead(ctx, uid, nil); err != nil {
		t.Fatalf("MarkRead all: %v", err)
	}
	if n, _ := repo.CountUnread(ctx, uid); n != 0 {
		t.Fatalf("после mark-all ожидал 0, получил %d", n)
	}

	// Prune с огромным сроком ничего не трогает (записи свежие).
	if n, err := repo.Prune(ctx, 3650); err != nil || n != 0 {
		t.Fatalf("Prune(3650): n=%d err=%v (ожидал 0)", n, err)
	}
}
