package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"efootball-bot/internal/repository"
	"efootball-bot/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestFriendlyRepoE2E — главный сценарий бага «нельзя вызвать друга повторно»:
// вызов → принятие → счёт → подтверждение → НОВЫЙ вызов той же паре должен
// проходить. Плюс отклонение/отмена/спор и TTL-истечение зависших матчей.
func TestFriendlyRepoE2E(t *testing.T) {
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

	newUser := func(tgID int64, name string) int64 {
		var id int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO users(telegram_id, display_name, created_at) VALUES ($1,$2,NOW()) RETURNING id`,
			tgID, name).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", name, err)
		}
		return id
	}
	alice := newUser(880001, "FriendlyAlice")
	bob := newUser(880002, "FriendlyBob")

	repo := repository.NewFriendlyRepository(pool)

	mustTransition := func(id int64, from, to string) {
		changed, err := repo.SetStatus(ctx, id, from, to)
		if err != nil || !changed {
			t.Fatalf("SetStatus %d %s→%s: changed=%v err=%v", id, from, to, changed, err)
		}
	}
	backdate := func(id int64, interval string) {
		if _, err := pool.Exec(ctx,
			`UPDATE friendlies SET updated_at = NOW() - $2::interval WHERE id = $1`, id, interval); err != nil {
			t.Fatalf("backdate %d: %v", id, err)
		}
	}

	// ── 1. Полный цикл и повторный вызов ──────────────────────────────
	f1, err := repo.Create(ctx, alice, bob)
	if err != nil {
		t.Fatalf("Create #1: %v", err)
	}
	if f1.Status != "pending" {
		t.Fatalf("новый вызов должен быть pending, got %q", f1.Status)
	}

	// Пока вызов активен — дубликат запрещён (в обе стороны).
	if _, err := repo.Create(ctx, alice, bob); !errors.Is(err, repository.ErrFriendlyActiveExists) {
		t.Fatalf("дубликат вызова должен вернуть ErrFriendlyActiveExists, got %v", err)
	}
	if _, err := repo.Create(ctx, bob, alice); !errors.Is(err, repository.ErrFriendlyActiveExists) {
		t.Fatalf("встречный вызов должен вернуть ErrFriendlyActiveExists, got %v", err)
	}

	mustTransition(f1.ID, "pending", "accepted")
	if changed, err := repo.ClaimScore(ctx, f1.ID, alice, 3, 1); err != nil || !changed {
		t.Fatalf("ClaimScore: changed=%v err=%v", changed, err)
	}
	if changed, err := repo.Confirm(ctx, f1.ID); err != nil || !changed {
		t.Fatalf("Confirm: changed=%v err=%v", changed, err)
	}

	// КЛЮЧЕВАЯ проверка: после подтверждения счёта пара свободна.
	f2, err := repo.Create(ctx, bob, alice)
	if err != nil {
		t.Fatalf("повторный вызов после confirmed должен проходить: %v", err)
	}

	// ── 2. Отклонение и отмена тоже освобождают пару ──────────────────
	mustTransition(f2.ID, "pending", "declined")
	f3, err := repo.Create(ctx, alice, bob)
	if err != nil {
		t.Fatalf("вызов после declined: %v", err)
	}
	mustTransition(f3.ID, "pending", "cancelled")
	f4, err := repo.Create(ctx, alice, bob)
	if err != nil {
		t.Fatalf("вызов после cancelled: %v", err)
	}

	// ── 3. Спорный счёт: reject → повторный ввод → confirm ────────────
	mustTransition(f4.ID, "pending", "accepted")
	if changed, err := repo.ClaimScore(ctx, f4.ID, alice, 5, 0); err != nil || !changed {
		t.Fatalf("ClaimScore до спора: changed=%v err=%v", changed, err)
	}
	mustTransition(f4.ID, "score_claimed", "accepted") // соперник оспорил
	if changed, err := repo.ClaimScore(ctx, f4.ID, bob, 2, 2); err != nil || !changed {
		t.Fatalf("повторный ClaimScore после спора: changed=%v err=%v", changed, err)
	}
	if changed, err := repo.Confirm(ctx, f4.ID); err != nil || !changed {
		t.Fatalf("Confirm после спора: changed=%v err=%v", changed, err)
	}

	// ── 4. Ленивое истечение в Create: зависший счёт не блокирует пару ─
	f5, err := repo.Create(ctx, alice, bob)
	if err != nil {
		t.Fatalf("Create #5: %v", err)
	}
	mustTransition(f5.ID, "pending", "accepted")
	if changed, err := repo.ClaimScore(ctx, f5.ID, alice, 1, 0); err != nil || !changed {
		t.Fatalf("ClaimScore #5: changed=%v err=%v", changed, err)
	}
	backdate(f5.ID, "3 days") // соперник так и не подтвердил

	f6, err := repo.Create(ctx, bob, alice)
	if err != nil {
		t.Fatalf("новый вызов должен вытеснить зависший score_claimed: %v", err)
	}
	if got, _ := repo.Get(ctx, f5.ID); got == nil || got.Status != "expired" {
		t.Fatalf("зависший матч должен стать expired, got %+v", got)
	}

	// Свежий матч ленивое истечение трогать не должно.
	if got, _ := repo.Get(ctx, f6.ID); got == nil || got.Status != "pending" {
		t.Fatalf("свежий вызов пострадал от очистки: %+v", got)
	}

	// ── 5. Фоновый свипер ExpireStale ──────────────────────────────────
	mustTransition(f6.ID, "pending", "accepted")
	backdate(f6.ID, "8 days") // приняли и не сыграли неделю
	refs, err := repo.ExpireStale(ctx)
	if err != nil {
		t.Fatalf("ExpireStale: %v", err)
	}
	found := false
	for _, ref := range refs {
		if ref.ID == f6.ID {
			found = true
			if ref.ChallengerID != bob || ref.OpponentID != alice {
				t.Fatalf("ExpireStale вернул неверных участников: %+v", ref)
			}
		}
	}
	if !found {
		t.Fatalf("ExpireStale не нашёл зависший матч %d (refs=%v)", f6.ID, refs)
	}

	// После истечения пара снова свободна.
	if _, err := repo.Create(ctx, alice, bob); err != nil {
		t.Fatalf("вызов после expired: %v", err)
	}

	// Истёкшие матчи видны в истории (скрываем только cancelled).
	list, err := repo.ListForUser(ctx, alice, 50)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	hasExpired := false
	for _, f := range list {
		if f.Status == "cancelled" {
			t.Fatalf("cancelled не должен попадать в список")
		}
		if f.Status == "expired" {
			hasExpired = true
		}
	}
	if !hasExpired {
		t.Fatal("expired-матчи должны оставаться в истории")
	}
}
