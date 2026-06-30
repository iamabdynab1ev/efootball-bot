package service

import (
	"context"
	"os"
	"testing"

	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestAdminScoreControlE2E проверяет ручной контроль счёта админом:
// установка → изменение → отмена → повторная установка, и что таблица всегда
// пересчитывается точно (без накопления дрейфа).
func TestAdminScoreControlE2E(t *testing.T) {
	dsn := os.Getenv("EFL_TEST_DSN")
	if dsn == "" {
		t.Skip("EFL_TEST_DSN не задан")
	}
	ctx := context.Background()
	testsupport.MigrateLocked(t, dsn, "../../migrations")

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	leagueRepo := repository.NewLeagueRepository(pool)
	matchRepo := repository.NewMatchRepository(pool)
	schedSvc := NewScheduleService(matchRepo, leagueRepo)
	matchSvc := NewMatchService(matchRepo, leagueRepo)

	season, err := leagueRepo.GetOrCreateActiveSeason(ctx)
	if err != nil {
		t.Fatalf("season: %v", err)
	}
	league, err := leagueRepo.CreateLeague(ctx, season.ID, "Admin Score League", nil, "single", 0, 0, 0)
	if err != nil {
		t.Fatalf("create league: %v", err)
	}

	// Два игрока → один матч.
	una, unb := "asc_a", "asc_b"
	ua, err := userRepo.Create(ctx, 960101, "ASC A", &una)
	if err != nil {
		t.Fatalf("user a: %v", err)
	}
	ub, err := userRepo.Create(ctx, 960102, "ASC B", &unb)
	if err != nil {
		t.Fatalf("user b: %v", err)
	}
	for _, id := range []int64{ua.ID, ub.ID} {
		if err := leagueRepo.AddMember(ctx, league.ID, id); err != nil {
			t.Fatalf("add member: %v", err)
		}
		if err := leagueRepo.ApproveMember(ctx, league.ID, id); err != nil {
			t.Fatalf("approve: %v", err)
		}
	}
	if err := leagueRepo.SetLeagueStatus(ctx, league.ID, "active"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if err := schedSvc.GenerateSchedule(ctx, league.ID, false); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	matches, err := matchRepo.GetAllForLeague(ctx, league.ID)
	if err != nil || len(matches) != 1 {
		t.Fatalf("ожидался 1 матч, n=%d err=%v", len(matches), err)
	}
	m := matches[0]
	home, away := m.HomeUserID, m.AwayUserID

	type s struct{ p, w, d, l, gf, ga int16 }
	check := func(label string, uid int64, want s) {
		ms, err := leagueRepo.GetMemberStats(ctx, league.ID, uid)
		if err != nil || ms == nil {
			t.Fatalf("%s: member stats uid=%d err=%v", label, uid, err)
		}
		got := s{ms.Points, ms.Wins, ms.Draws, ms.Losses, ms.GoalsFor, ms.GoalsAgainst}
		if got != want {
			t.Fatalf("%s uid=%d: got %+v, want %+v", label, uid, got, want)
		}
	}

	// 1) Установка 3:1 — хозяин побеждает.
	if _, err := matchSvc.AdminSetScore(ctx, m.ID, 3, 1); err != nil {
		t.Fatalf("set 3:1: %v", err)
	}
	check("set", home, s{3, 1, 0, 0, 3, 1})
	check("set", away, s{0, 0, 0, 1, 1, 3})

	// 2) Изменение на 0:0 — НЕ должно складываться с прошлым (точный пересчёт).
	if _, err := matchSvc.AdminSetScore(ctx, m.ID, 0, 0); err != nil {
		t.Fatalf("change 0:0: %v", err)
	}
	check("change", home, s{1, 0, 1, 0, 0, 0})
	check("change", away, s{1, 0, 1, 0, 0, 0})

	// 3) Отмена — статистика обнуляется, матч возвращается в scheduled.
	cancelled, err := matchSvc.AdminCancelScore(ctx, m.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Status != models.MatchScheduled || cancelled.HomeGoals != nil || cancelled.AwayGoals != nil {
		t.Fatalf("после отмены матч не сброшен: status=%s goals=%v/%v", cancelled.Status, cancelled.HomeGoals, cancelled.AwayGoals)
	}
	check("cancel", home, s{0, 0, 0, 0, 0, 0})
	check("cancel", away, s{0, 0, 0, 0, 0, 0})

	// 4) Повторная установка 2:2 поверх отменённого — снова точно.
	if _, err := matchSvc.AdminSetScore(ctx, m.ID, 2, 2); err != nil {
		t.Fatalf("re-set 2:2: %v", err)
	}
	check("re-set", home, s{1, 0, 1, 0, 2, 2})
	check("re-set", away, s{1, 0, 1, 0, 2, 2})

	t.Log("✅ Админ set/change/cancel/re-set: таблица пересчитывается точно, без дрейфа")
}
