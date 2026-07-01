package service

import (
	"context"
	"errors"
	"os"
	"testing"

	"efootball-bot/internal/models"
	"efootball-bot/internal/repository"
	"efootball-bot/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestChatE2E проверяет чат: членство по группам, отправку с fan-out, догрузку
// (since/before, no-loss), архивацию и админ-удаление.
func TestChatE2E(t *testing.T) {
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
	chatRepo := repository.NewChatRepository(pool)

	// fan-out перехватываем, чтобы проверить адресную доставку участникам.
	var lastFanout []int64
	chatSvc := NewChatService(chatRepo, leagueRepo, func(ids []int64, _ string, _ any) {
		lastFanout = ids
	})

	season, _ := leagueRepo.GetOrCreateActiveSeason(ctx)
	league, err := leagueRepo.CreateLeague(ctx, season.ID, "Chat League", nil, "groups", 2, 1, 0)
	if err != nil {
		t.Fatalf("league: %v", err)
	}

	mk := func(tg int64, name string) int64 {
		un := name
		u, err := userRepo.Create(ctx, tg, name, &un)
		if err != nil {
			t.Fatalf("user %s: %v", name, err)
		}
		if err := leagueRepo.AddMember(ctx, league.ID, u.ID); err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
		if err := leagueRepo.ApproveMember(ctx, league.ID, u.ID); err != nil {
			t.Fatalf("approve %s: %v", name, err)
		}
		return u.ID
	}
	a1 := mk(950201, "Akmal")
	a2 := mk(950202, "Aziz")
	b1 := mk(950203, "Bek")
	outsiderName := "outsider"
	outsider, _ := userRepo.Create(ctx, 950204, "Outsider", &outsiderName) // НЕ в лиге

	// Группы: a1,a2 → A; b1 → B.
	if err := leagueRepo.SetMemberGroups(ctx, league.ID, []int64{a1, a2, b1}, []string{"A", "A", "B"}); err != nil {
		t.Fatalf("set groups: %v", err)
	}

	// Комнаты для a1: общая + A.
	rooms, err := chatSvc.RoomsForUser(ctx, a1, league.ID)
	if err != nil {
		t.Fatalf("rooms a1: %v", err)
	}
	var roomA, roomB, roomGeneral int64
	for _, r := range rooms {
		switch r.GroupName {
		case "A":
			roomA = r.ID
		case "":
			roomGeneral = r.ID
		}
	}
	if roomA == 0 || roomGeneral == 0 {
		t.Fatalf("a1 должен видеть общую и A комнаты, получил %+v", rooms)
	}
	if len(rooms) != 2 {
		t.Fatalf("a1 ожидал 2 комнаты (общая+A), получил %d", len(rooms))
	}
	// Найдём комнату B через админ-список.
	all, _ := chatSvc.ListRooms(ctx, league.ID)
	for _, r := range all {
		if r.GroupName == "B" {
			roomB = r.ID
		}
	}
	if roomB == 0 {
		t.Fatal("комната B не создана")
	}

	// Скоуп @упоминаний по комнате: общая = вся лига (3), A = только a1,a2 (2), B = 1.
	genM, err := chatSvc.Members(ctx, a1, roomGeneral)
	if err != nil || len(genM) != 3 {
		t.Fatalf("участники общей комнаты: n=%d err=%v (ждали 3)", len(genM), err)
	}
	aM, err := chatSvc.Members(ctx, a1, roomA)
	if err != nil || len(aM) != 2 {
		t.Fatalf("участники A: n=%d err=%v (ждали 2)", len(aM), err)
	}
	bM, err := chatSvc.Members(ctx, b1, roomB)
	if err != nil || len(bM) != 1 {
		t.Fatalf("участники B: n=%d err=%v (ждали 1)", len(bM), err)
	}
	// b1 не может смотреть участников A (нет доступа).
	if _, err := chatSvc.Members(ctx, b1, roomA); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("b1 не должен видеть участников A: err=%v", err)
	}

	// a1 пишет в A → fan-out участникам A (a1,a2), но не b1.
	msg, err := chatSvc.Send(ctx, a1, roomA, "  привет группа A  ")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if msg.Body != "привет группа A" || msg.AuthorName != "Akmal" {
		t.Fatalf("сообщение некорректно: %+v", msg)
	}
	got := map[int64]bool{}
	for _, id := range lastFanout {
		got[id] = true
	}
	if !got[a1] || !got[a2] || got[b1] {
		t.Fatalf("fan-out по A неверный: %v (ждали a1,a2 без b1)", lastFanout)
	}

	// b1 НЕ может писать в A.
	if _, err := chatSvc.Send(ctx, b1, roomA, "я из B"); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("b1 не должен писать в A: err=%v", err)
	}
	// Аутсайдер не может писать в общую.
	if _, err := chatSvc.Send(ctx, outsider.ID, roomGeneral, "чужой"); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("аутсайдер не должен писать: err=%v", err)
	}
	// Пустое сообщение.
	if _, err := chatSvc.Send(ctx, a1, roomA, "   "); !errors.Is(err, ErrChatEmpty) {
		t.Fatalf("пустое сообщение должно отклоняться: err=%v", err)
	}

	// История для a2 (член A) видит сообщение; since=msg.ID → пусто (нет новее).
	hist, err := chatSvc.History(ctx, a2, roomA, 0, 0, 50)
	if err != nil || len(hist) != 1 || hist[0].ID != msg.ID {
		t.Fatalf("история A: n=%d err=%v", len(hist), err)
	}
	since, err := chatSvc.History(ctx, a2, roomA, 0, msg.ID, 50)
	if err != nil || len(since) != 0 {
		t.Fatalf("since должен быть пуст: n=%d err=%v", len(since), err)
	}
	// b1 не может читать историю A.
	if _, err := chatSvc.History(ctx, b1, roomA, 0, 0, 50); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("b1 не должен читать A: err=%v", err)
	}

	// Админ-удаление → тело скрыто, помечено deleted.
	del, err := chatSvc.DeleteMessage(ctx, msg.ID)
	if err != nil || !del.Deleted || del.Body != "" {
		t.Fatalf("удаление: %+v err=%v", del, err)
	}
	after, _ := chatSvc.History(ctx, a2, roomA, 0, 0, 50)
	if len(after) != 1 || !after[0].Deleted || after[0].Body != "" {
		t.Fatalf("после удаления тело должно быть скрыто: %+v", after)
	}

	// @упоминание: a1 упоминает @Aziz в комнате A → onMention только с a2 (не автор).
	var mentioned []int64
	var mentionLeague int64
	chatSvc.SetMentionHandler(func(_ context.Context, _ *models.ChatMessage, ids []int64, leagueID int64) {
		mentioned = ids
		mentionLeague = leagueID
	})
	if _, err := chatSvc.Send(ctx, a1, roomA, "эй @Aziz глянь счёт, @Akmal тоже"); err != nil {
		t.Fatalf("send with mention: %v", err)
	}
	if len(mentioned) != 1 || mentioned[0] != a2 {
		t.Fatalf("упоминание должно указывать только на a2 (не автора a1): %v", mentioned)
	}
	if mentionLeague != league.ID {
		t.Fatalf("league в упоминании неверен: %d != %d", mentionLeague, league.ID)
	}
	// Упоминание игрока не из этой комнаты (Bek в группе B) не срабатывает.
	mentioned = nil
	if _, err := chatSvc.Send(ctx, a1, roomA, "@Bek сюда нельзя"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(mentioned) != 0 {
		t.Fatalf("Bek не участник A — упоминание не должно срабатывать: %v", mentioned)
	}
	chatSvc.SetMentionHandler(nil)

	// Архивация → отправка запрещена.
	if err := chatSvc.Archive(ctx, league.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if _, err := chatSvc.Send(ctx, a1, roomA, "после архива"); !errors.Is(err, ErrChatArchived) {
		t.Fatalf("в архивный чат писать нельзя: err=%v", err)
	}

	t.Log("✅ Чат: членство по группам, fan-out, no-loss догрузка, удаление, архивация")
}
