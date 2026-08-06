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

// TestFinalizeChampionIsFinalWinner проверяет ключевую корректность выдачи кубка:
// в формате с плей-офф чемпион = ПОБЕДИТЕЛЬ ФИНАЛА, даже если ручная
// «Финализировать лигу» (FinalizeLeague без явного чемпиона) вызвана, и даже
// если лидер групповой таблицы — другой игрок. Серебро — проигравшему финала.
func TestFinalizeChampionIsFinalWinner(t *testing.T) {
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

	userRepo := repository.NewUserRepository(pool)
	leagueRepo := repository.NewLeagueRepository(pool)
	matchRepo := repository.NewMatchRepository(pool)
	achievRepo := repository.NewAchievementRepository(pool)
	awardRepo := repository.NewAwardRepository(pool)
	awardSvc := NewAwardService(awardRepo, leagueRepo, achievRepo, matchRepo)

	season, err := leagueRepo.GetOrCreateActiveSeason(ctx)
	if err != nil {
		t.Fatalf("season: %v", err)
	}
	league, err := leagueRepo.CreateLeague(ctx, season.ID, "Finalize Champion Test", nil, "groups_playoff", 0, 0, 0)
	if err != nil {
		t.Fatalf("create league: %v", err)
	}

	// Два игрока: A — лидер таблицы, B — победитель финала.
	unameA, unameB := "champA", "champB"
	ua, err := userRepo.Create(ctx, 950001, "Table Leader A", &unameA)
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	ub, err := userRepo.Create(ctx, 950002, "Final Winner B", &unameB)
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	for _, uid := range []int64{ua.ID, ub.ID} {
		if err := leagueRepo.AddMember(ctx, league.ID, uid); err != nil {
			t.Fatalf("add member: %v", err)
		}
		if err := leagueRepo.ApproveMember(ctx, league.ID, uid); err != nil {
			t.Fatalf("approve: %v", err)
		}
	}

	// Групповой матч: A разгромил B 3:0 — теперь A выше в таблице (лидер).
	if _, err := pool.Exec(ctx, `
		INSERT INTO matches (league_id, home_user_id, away_user_id, home_goals, away_goals, status, stage, round)
		VALUES ($1, $2, $3, 3, 0, 'confirmed'::match_status, 'A', 1)
	`, league.ID, ua.ID, ub.ID); err != nil {
		t.Fatalf("insert group match: %v", err)
	}
	if err := leagueRepo.RecalculateTable(ctx, league.ID); err != nil {
		t.Fatalf("recalc table: %v", err)
	}
	if err := RecalculatePositionsH2H(ctx, league.ID, leagueRepo, matchRepo); err != nil {
		t.Fatalf("recalc positions: %v", err)
	}

	// ФИНАЛ: B обыграл A 1:0 (B — гость). Чемпион турнира — B, не лидер таблицы A.
	if _, err := pool.Exec(ctx, `
		INSERT INTO matches (league_id, home_user_id, away_user_id, home_goals, away_goals, status, stage, round, bracket_slot)
		VALUES ($1, $2, $3, 0, 1, 'confirmed'::match_status, 'final', 100, 1)
	`, league.ID, ua.ID, ub.ID); err != nil {
		t.Fatalf("insert final: %v", err)
	}

	// Ручная финализация БЕЗ явного чемпиона — как кнопка «Финализировать лигу».
	if err := awardSvc.FinalizeLeague(ctx, league.ID); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	// Проверяем награды именно ЭТОЙ лиги (изоляция от прочих прогонов в общей БД).
	hasAward := func(uid int64, awardType string) bool {
		var cnt int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM season_awards WHERE league_id=$1 AND user_id=$2 AND award_type=$3`,
			league.ID, uid, awardType).Scan(&cnt); err != nil {
			t.Fatalf("query award: %v", err)
		}
		return cnt > 0
	}

	if !hasAward(ub.ID, "champion") {
		t.Fatal("КУБОК НЕ ТОМУ: чемпионом должен быть победитель финала B, но у B нет трофея champion")
	}
	if hasAward(ua.ID, "champion") {
		t.Fatal("КУБОК НЕ ТОМУ: лидер таблицы A получил champion, хотя проиграл финал")
	}
	if !hasAward(ua.ID, "runner_up") {
		t.Fatal("серебро должно уйти проигравшему финала A")
	}
	_ = models.StageFinal
}
