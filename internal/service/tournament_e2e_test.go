package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"efootball-bot/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// TestTournamentE2E — сквозной тест полного турнира против реальной БД.
// Запускается только когда задан EFL_TEST_DSN (локальный postgres), иначе
// пропускается — чтобы обычный `go test ./...` не требовал БД.
func TestTournamentE2E(t *testing.T) {
	dsn := os.Getenv("EFL_TEST_DSN")
	if dsn == "" {
		t.Skip("EFL_TEST_DSN не задан — пропускаю интеграционный тест")
	}
	ctx := context.Background()

	// ── Миграции ──
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	goose.SetDialect("postgres")
	if err := goose.Up(sqlDB, "../../migrations"); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	sqlDB.Close()

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

	// ── Сезон + лига (один круг) ──
	season, err := leagueRepo.GetOrCreateActiveSeason(ctx)
	if err != nil {
		t.Fatalf("season: %v", err)
	}
	league, err := leagueRepo.CreateLeague(ctx, season.ID, "E2E Test League", nil, "single", 0, 0, 0)
	if err != nil {
		t.Fatalf("create league: %v", err)
	}

	// ── 4 игрока ──
	const n = 4
	uids := make([]int64, n)
	for i := 0; i < n; i++ {
		uname := fmt.Sprintf("p%d", i+1)
		u, err := userRepo.Create(ctx, int64(900000+i), fmt.Sprintf("Player %d", i+1), &uname)
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
		uids[i] = u.ID
		if err := leagueRepo.AddMember(ctx, league.ID, u.ID); err != nil {
			t.Fatalf("add member: %v", err)
		}
		if err := leagueRepo.ApproveMember(ctx, league.ID, u.ID); err != nil {
			t.Fatalf("approve: %v", err)
		}
	}
	if err := leagueRepo.SetLeagueStatus(ctx, league.ID, "active"); err != nil {
		t.Fatalf("activate: %v", err)
	}

	// ── Жеребьёвка ──
	if err := schedSvc.GenerateSchedule(ctx, league.ID, false); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	matches, err := matchRepo.GetAllForLeague(ctx, league.ID)
	if err != nil {
		t.Fatalf("get matches: %v", err)
	}

	// 4 игрока, один круг → C(4,2) = 6 матчей; каждая пара ровно один раз
	if len(matches) != 6 {
		t.Fatalf("ожидалось 6 матчей, получено %d", len(matches))
	}
	pairSeen := map[string]int{}
	for _, m := range matches {
		a, b := m.HomeUserID, m.AwayUserID
		if a > b {
			a, b = b, a
		}
		pairSeen[fmt.Sprintf("%d-%d", a, b)]++
	}
	if len(pairSeen) != 6 {
		t.Fatalf("ожидалось 6 уникальных пар, получено %d", len(pairSeen))
	}
	for k, c := range pairSeen {
		if c != 1 {
			t.Fatalf("пара %s сыграна %d раз (ожидалось 1)", k, c)
		}
	}

	// ── Применяем результаты (как в проде: ClaimResult → Confirm) ──
	type st struct{ p, w, d, l, gf, ga int }
	exp := map[int64]*st{}
	for _, id := range uids {
		exp[id] = &st{}
	}
	apply := func(m *modelsMatch, hg, ag int16) {
		if err := matchRepo.ClaimResult(ctx, m.id, hg, ag); err != nil {
			t.Fatalf("claim: %v", err)
		}
		if _, err := matchSvc.Confirm(ctx, m.id); err != nil {
			t.Fatalf("confirm: %v", err)
		}
		h, a := exp[m.home], exp[m.away]
		h.gf += int(hg); h.ga += int(ag); a.gf += int(ag); a.ga += int(hg)
		switch {
		case hg > ag:
			h.w++; h.p += 3; a.l++
		case hg < ag:
			a.w++; a.p += 3; h.l++
		default:
			h.d++; h.p++; a.d++; a.p++
		}
	}

	// детерминированные счёта для каждого матча
	for i, m := range matches {
		mm := &modelsMatch{id: m.ID, home: m.HomeUserID, away: m.AwayUserID}
		hg := int16((i % 3) + 1) // 1,2,3,1,2,3
		ag := int16(i % 2)       // 0,1,0,1,0,1
		apply(mm, hg, ag)
	}

	// ── Проверяем таблицу ──
	members, err := leagueRepo.GetMembers(ctx, league.ID)
	if err != nil {
		t.Fatalf("get members: %v", err)
	}
	for _, mb := range members {
		e := exp[mb.UserID]
		if int(mb.Points) != e.p || int(mb.Wins) != e.w || int(mb.Draws) != e.d || int(mb.Losses) != e.l ||
			int(mb.GoalsFor) != e.gf || int(mb.GoalsAgainst) != e.ga {
			t.Fatalf("игрок %d: таблица БД {P%d W%d D%d L%d GF%d GA%d} != ожидание {P%d W%d D%d L%d GF%d GA%d}",
				mb.UserID, mb.Points, mb.Wins, mb.Draws, mb.Losses, mb.GoalsFor, mb.GoalsAgainst,
				e.p, e.w, e.d, e.l, e.gf, e.ga)
		}
	}

	// все 6 матчей подтверждены
	confirmed := 0
	allM, _ := matchRepo.GetAllForLeague(ctx, league.ID)
	for _, m := range allM {
		if m.Status == "confirmed" {
			confirmed++
		}
	}
	if confirmed != 6 {
		t.Fatalf("подтверждено %d матчей, ожидалось 6", confirmed)
	}

	// ── Позиции в таблице: уникальны 1..n и не растут по очкам ──
	posSet := map[int16]bool{}
	var prevPts int16 = 1 << 14
	for _, mb := range members {
		if mb.Position == nil {
			t.Fatalf("игрок %d: позиция не проставлена", mb.UserID)
		}
		p := *mb.Position
		if p < 1 || p > int16(n) || posSet[p] {
			t.Fatalf("некорректная/дублирующая позиция %d", p)
		}
		posSet[p] = true
		if mb.Points > prevPts {
			t.Fatalf("порядок таблицы нарушен: позиция %d имеет больше очков (%d > %d)", p, mb.Points, prevPts)
		}
		prevPts = mb.Points
	}

	// ── Повторная жеребьёвка должна быть отклонена ──
	if err := schedSvc.GenerateSchedule(ctx, league.ID, false); err == nil {
		t.Fatal("повторная жеребьёвка должна быть отклонена, но прошла")
	}

	t.Logf("✅ Один круг: 6 матчей, таблица и позиции сходятся, повторная жеребьёвка отклонена")

	// ── Двойной круг: каждая пара играет дважды (12 матчей) ──
	league2, err := leagueRepo.CreateLeague(ctx, season.ID, "E2E Double", nil, "double", 0, 0, 0)
	if err != nil {
		t.Fatalf("create league2: %v", err)
	}
	for _, id := range uids {
		if err := leagueRepo.AddMember(ctx, league2.ID, id); err != nil {
			t.Fatalf("add member2: %v", err)
		}
		if err := leagueRepo.ApproveMember(ctx, league2.ID, id); err != nil {
			t.Fatalf("approve2: %v", err)
		}
	}
	if err := schedSvc.GenerateSchedule(ctx, league2.ID, true); err != nil {
		t.Fatalf("schedule2: %v", err)
	}
	m2, _ := matchRepo.GetAllForLeague(ctx, league2.ID)
	if len(m2) != 12 {
		t.Fatalf("двойной круг: ожидалось 12 матчей, получено %d", len(m2))
	}
	pair2 := map[string]int{}
	for _, m := range m2 {
		a, b := m.HomeUserID, m.AwayUserID
		if a > b {
			a, b = b, a
		}
		pair2[fmt.Sprintf("%d-%d", a, b)]++
	}
	for k, c := range pair2 {
		if c != 2 {
			t.Fatalf("двойной круг: пара %s сыграна %d раз (ожидалось 2)", k, c)
		}
	}
	t.Logf("✅ Двойной круг: 12 матчей, каждая пара дважды")
}

type modelsMatch struct {
	id, home, away int64
}
