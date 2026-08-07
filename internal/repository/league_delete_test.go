package repository

import (
	"context"
	"os"
	"testing"

	"efootball-bot/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestDeleteLeagueWithConflictingAwards ловит реальный баг: удаление лиги
// падало с нарушением уникальности, когда у игрока была И глобальная, И лиговая
// версия одного достижения (или лиговый + сезонный трофей одного типа) — авто
// ON DELETE SET NULL создавал дубликат по частичному уникальному индексу.
func TestDeleteLeagueWithConflictingAwards(t *testing.T) {
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

	userRepo := NewUserRepository(pool)
	leagueRepo := NewLeagueRepository(pool)
	achievRepo := NewAchievementRepository(pool)
	awardRepo := NewAwardRepository(pool)

	season, err := leagueRepo.GetOrCreateActiveSeason(ctx)
	if err != nil {
		t.Fatalf("season: %v", err)
	}
	league, err := leagueRepo.CreateLeague(ctx, season.ID, "Delete Conflict Test", nil, "groups_playoff", 0, 0, 0)
	if err != nil {
		t.Fatalf("create league: %v", err)
	}

	uname := "delconf"
	u, err := userRepo.Create(ctx, 970001, "Del Conflict User", &uname)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Достижение 'scorer_10': глобальная (league_id NULL) И лиговая версии —
	// именно эта пара порождала конфликт при SET NULL.
	if _, err := achievRepo.Award(ctx, u.ID, "scorer_10", nil); err != nil {
		t.Fatalf("award global: %v", err)
	}
	if _, err := achievRepo.Award(ctx, u.ID, "scorer_10", &league.ID); err != nil {
		t.Fatalf("award league: %v", err)
	}

	// Трофей 'champion': лиговый И сезонный (league_id NULL) — вторая грань бага.
	if _, err := awardRepo.CreateAward(ctx, season.ID, league.ID, "champion", u.ID, 0); err != nil {
		t.Fatalf("create league award: %v", err)
	}
	if _, err := awardRepo.CreateSeasonAward(ctx, season.ID, "champion", u.ID, 0); err != nil {
		t.Fatalf("create season award: %v", err)
	}

	// Полное удаление лиги — раньше здесь была ошибка уникальности.
	if err := leagueRepo.DeleteLeague(ctx, league.ID); err != nil {
		t.Fatalf("НЕ УДАЛОСЬ УДАЛИТЬ ЛИГУ (регресс бага): %v", err)
	}

	l, err := leagueRepo.GetByID(ctx, league.ID)
	if err != nil {
		t.Fatalf("GetByID after delete: %v", err)
	}
	if l != nil {
		t.Fatal("лига должна быть удалена, но всё ещё существует")
	}
}
