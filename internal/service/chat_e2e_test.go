package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

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
	msg, err := chatSvc.Send(ctx, a1, roomA, "  привет группа A  ", nil)
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
	if _, err := chatSvc.Send(ctx, b1, roomA, "я из B", nil); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("b1 не должен писать в A: err=%v", err)
	}
	// Аутсайдер не может писать в общую.
	if _, err := chatSvc.Send(ctx, outsider.ID, roomGeneral, "чужой", nil); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("аутсайдер не должен писать: err=%v", err)
	}
	// Пустое сообщение.
	if _, err := chatSvc.Send(ctx, a1, roomA, "   ", nil); !errors.Is(err, ErrChatEmpty) {
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

	// Групповые отметки прочтения: a2 читает сообщение a1 в комнате A.
	if _, err := chatSvc.MarkRead(ctx, a2, roomA, msg.ID); err != nil {
		t.Fatalf("MarkRead a2 в A: %v", err)
	}
	reads, err := chatSvc.RoomReads(ctx, a1, roomA)
	if err != nil {
		t.Fatalf("RoomReads A: %v", err)
	}
	var a2read int64 = -1
	for _, rr := range reads {
		if rr.UserID == a2 {
			a2read = rr.LastRead
		}
	}
	if a2read != msg.ID {
		t.Fatalf("a2 должен был прочитать до %d в A, got %d", msg.ID, a2read)
	}
	// b1 (не в группе A) не может получить прогресс прочтения A.
	if _, err := chatSvc.RoomReads(ctx, b1, roomA); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("b1 не должен видеть reads A: err=%v", err)
	}
	// После прочтения непрочитанных у a2 в комнате A нет.
	roomsA2, _ := chatSvc.RoomsForUser(ctx, a2, league.ID)
	for _, rm := range roomsA2 {
		if rm.ID == roomA && rm.Unread != 0 {
			t.Fatalf("у a2 непрочитанных в A должно быть 0, got %d", rm.Unread)
		}
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

	// Правка и удаление СВОИХ сообщений (пользователь, не админ).
	own, err := chatSvc.Send(ctx, a1, roomA, "переиграем?", nil)
	if err != nil {
		t.Fatalf("send own: %v", err)
	}
	ed, err := chatSvc.EditMessage(ctx, a1, own.ID, "переиграем в 20:00")
	if err != nil || !ed.Edited || ed.Body != "переиграем в 20:00" {
		t.Fatalf("правка своего: %+v err=%v", ed, err)
	}
	if _, err := chatSvc.EditMessage(ctx, a2, own.ID, "чужое"); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("нельзя править чужое сообщение: err=%v", err)
	}
	if _, err := chatSvc.DeleteOwnMessage(ctx, a2, false, own.ID); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("нельзя удалять чужое без прав: err=%v", err)
	}
	delOwn, err := chatSvc.DeleteOwnMessage(ctx, a1, false, own.ID)
	if err != nil || !delOwn.Deleted {
		t.Fatalf("удаление своего: %+v err=%v", delOwn, err)
	}

	// Ответ на сообщение (reply_to_id).
	base2, err := chatSvc.Send(ctx, a1, roomA, "го на матч", nil)
	if err != nil {
		t.Fatalf("send base2: %v", err)
	}
	rep, err := chatSvc.Send(ctx, a2, roomA, "ок", &base2.ID)
	if err != nil || rep.ReplyToID == nil || *rep.ReplyToID != base2.ID {
		t.Fatalf("reply_to_id не сохранён: %+v err=%v", rep, err)
	}
	// Ответ на сообщение из ДРУГОЙ комнаты игнорируется (reply_to_id обнуляется).
	gmsg, _ := chatSvc.Send(ctx, a1, roomGeneral, "в общем", nil)
	bad, _ := chatSvc.Send(ctx, a1, roomA, "ответ на чужую комнату", &gmsg.ID)
	if bad.ReplyToID != nil {
		t.Fatalf("reply на сообщение чужой комнаты должен обнулиться: %+v", bad)
	}

	// Реакции: a2 ставит 👍 на base2 → агрегат 1, mine=true.
	if err := chatSvc.AddReaction(ctx, a2, base2.ID, "👍"); err != nil {
		t.Fatalf("add reaction: %v", err)
	}
	rx, _ := chatSvc.RoomReactions(ctx, a2, roomA)
	foundRx := false
	for _, x := range rx {
		if x.MessageID == base2.ID && x.Emoji == "👍" {
			if x.Count != 1 || !x.Mine {
				t.Fatalf("реакция-агрегат неверный: %+v", x)
			}
			foundRx = true
		}
	}
	if !foundRx {
		t.Fatalf("реакция не найдена в агрегате")
	}
	// b1 (не в группе A) не может реагировать.
	if err := chatSvc.AddReaction(ctx, b1, base2.ID, "🔥"); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("b1 не должен реагировать в A: err=%v", err)
	}
	// «Одна реакция на пользователя»: a2 меняет 👍 на ❤️ — старая снимается сама.
	if err := chatSvc.AddReaction(ctx, a2, base2.ID, "❤️"); err != nil {
		t.Fatalf("replace reaction: %v", err)
	}
	rxR, _ := chatSvc.RoomReactions(ctx, a2, roomA)
	var likes, hearts int
	for _, x := range rxR {
		if x.MessageID == base2.ID && x.Emoji == "👍" {
			likes += x.Count
		}
		if x.MessageID == base2.ID && x.Emoji == "❤️" {
			hearts += x.Count
			if !x.Mine {
				t.Fatalf("❤️ должна быть mine: %+v", x)
			}
		}
	}
	if likes != 0 || hearts != 1 {
		t.Fatalf("замена реакции: 👍=%d (ждали 0), ❤️=%d (ждали 1)", likes, hearts)
	}
	// Возвращаем 👍 обратно, чтобы дальнейший сценарий снятия работал как раньше.
	if err := chatSvc.AddReaction(ctx, a2, base2.ID, "👍"); err != nil {
		t.Fatalf("re-add reaction: %v", err)
	}
	// Снятие реакции.
	if err := chatSvc.RemoveReaction(ctx, a2, base2.ID, "👍"); err != nil {
		t.Fatalf("remove reaction: %v", err)
	}
	rx2, _ := chatSvc.RoomReactions(ctx, a2, roomA)
	for _, x := range rx2 {
		if x.MessageID == base2.ID && x.Emoji == "👍" {
			t.Fatalf("реакция должна быть снята, но осталась: %+v", x)
		}
	}

	// @упоминание: a1 упоминает @Aziz в комнате A → onMention только с a2 (не автор).
	var mentioned []int64
	var mentionLeague int64
	chatSvc.SetMentionHandler(func(_ context.Context, _ *models.ChatMessage, ids []int64, leagueID int64) {
		mentioned = ids
		mentionLeague = leagueID
	})
	if _, err := chatSvc.Send(ctx, a1, roomA, "эй @Aziz глянь счёт, @Akmal тоже", nil); err != nil {
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
	if _, err := chatSvc.Send(ctx, a1, roomA, "@Bek сюда нельзя", nil); err != nil {
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
	if _, err := chatSvc.Send(ctx, a1, roomA, "после архива", nil); !errors.Is(err, ErrChatArchived) {
		t.Fatalf("в архивный чат писать нельзя: err=%v", err)
	}

	t.Log("✅ Чат: членство по группам, fan-out, no-loss догрузка, удаление, архивация")
}

// TestChatDirectE2E проверяет личные сообщения соперникам: гейт по матчу,
// идемпотентность комнаты (нормализованная пара), доступ, fan-out и уведомление
// собеседнику, список диалогов.
func TestChatDirectE2E(t *testing.T) {
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

	var lastFanout []int64
	chatSvc := NewChatService(chatRepo, leagueRepo, func(ids []int64, _ string, _ any) {
		lastFanout = ids
	})
	var directRecipient int64
	chatSvc.SetDirectHandler(func(_ context.Context, _ *models.ChatMessage, recipientID int64) {
		directRecipient = recipientID
	})

	season, _ := leagueRepo.GetOrCreateActiveSeason(ctx)
	league, err := leagueRepo.CreateLeague(ctx, season.ID, "DM League", nil, "groups", 2, 1, 0)
	if err != nil {
		t.Fatalf("league: %v", err)
	}

	// Уникальные tg на каждый прогон: ЛС-комната ключуется парой пользователей и
	// сохраняется между запусками, поэтому нужны свежие пользователи для изоляции.
	base := time.Now().UnixNano() % 1_000_000_000
	mkUser := func(off int64, name string) int64 {
		un := name
		u, err := userRepo.Create(ctx, base+off, name, &un)
		if err != nil {
			t.Fatalf("user %s: %v", name, err)
		}
		return u.ID
	}
	p1 := mkUser(1, "Rustam")
	p2 := mkUser(2, "Sardor")
	stranger := mkUser(3, "Stranger")

	// p1 и p2 — соперники по матчу; stranger ни с кем не играл.
	if _, err := pool.Exec(ctx,
		`INSERT INTO matches (league_id, home_user_id, away_user_id, round) VALUES ($1,$2,$3,1)`,
		league.ID, p1, p2); err != nil {
		t.Fatalf("seed match: %v", err)
	}

	// Открытие ЛС с соперником — успех; kind=direct.
	room, err := chatSvc.OpenDirect(ctx, p1, p2)
	if err != nil || room == nil || room.Kind != "direct" {
		t.Fatalf("OpenDirect p1→p2: room=%+v err=%v", room, err)
	}
	// Идемпотентность: обратный порядок даёт ту же комнату (нормализованная пара).
	room2, err := chatSvc.OpenDirect(ctx, p2, p1)
	if err != nil || room2.ID != room.ID {
		t.Fatalf("OpenDirect p2→p1 должен вернуть ту же комнату: %+v vs %+v err=%v", room2, room, err)
	}
	// Нельзя писать тому, с кем не было матча.
	if _, err := chatSvc.OpenDirect(ctx, p1, stranger); !errors.Is(err, ErrChatNotOpponents) {
		t.Fatalf("OpenDirect к не-сопернику должен падать: err=%v", err)
	}
	// Нельзя открыть ЛС с самим собой.
	if _, err := chatSvc.OpenDirect(ctx, p1, p1); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("OpenDirect самому себе должен падать: err=%v", err)
	}

	// Отправка: fan-out обоим участникам, уведомление — собеседнику (p2).
	msg, err := chatSvc.Send(ctx, p1, room.ID, "во сколько играем?", nil)
	if err != nil {
		t.Fatalf("send direct: %v", err)
	}
	got := map[int64]bool{}
	for _, id := range lastFanout {
		got[id] = true
	}
	if !got[p1] || !got[p2] || len(lastFanout) != 2 {
		t.Fatalf("fan-out ЛС неверный: %v (ждали p1,p2)", lastFanout)
	}
	if directRecipient != p2 {
		t.Fatalf("уведомление должно уйти собеседнику p2, got=%d", directRecipient)
	}

	// Доступ: собеседник видит историю, посторонний — нет.
	hist, err := chatSvc.History(ctx, p2, room.ID, 0, 0, 50)
	if err != nil || len(hist) != 1 || hist[0].ID != msg.ID {
		t.Fatalf("история ЛС для p2: n=%d err=%v", len(hist), err)
	}
	if _, err := chatSvc.History(ctx, stranger, room.ID, 0, 0, 50); !errors.Is(err, ErrChatForbidden) {
		t.Fatalf("посторонний не должен читать ЛС: err=%v", err)
	}

	// Список диалогов p1: одна беседа с p2 и последним сообщением.
	convs, err := chatSvc.ListDirect(ctx, p1)
	if err != nil || len(convs) != 1 {
		t.Fatalf("ListDirect p1: n=%d err=%v", len(convs), err)
	}
	if convs[0].OtherID != p2 || convs[0].LastBody != "во сколько играем?" {
		t.Fatalf("диалог p1 некорректен: %+v", convs[0])
	}

	// Непрочитанные: у p2 одно непрочитанное (сообщение p1), у p1 — ноль (своё).
	if total, _ := chatSvc.UnreadTotal(ctx, p2); total != 1 {
		t.Fatalf("p2 должен иметь 1 непрочитанное, got=%d", total)
	}
	if total, _ := chatSvc.UnreadTotal(ctx, p1); total != 0 {
		t.Fatalf("p1 не должен иметь непрочитанных, got=%d", total)
	}
	p2convs, _ := chatSvc.ListDirect(ctx, p2)
	if len(p2convs) != 1 || p2convs[0].Unread != 1 {
		t.Fatalf("диалог p2: unread=%d (ждали 1)", p2convs[0].Unread)
	}

	// p2 читает до msg.ID → его непрочитанные обнуляются, а у p1 в диалоге
	// появляется other_last_read == msg.ID (для ✓✓).
	if _, err := chatSvc.MarkRead(ctx, p2, room.ID, msg.ID); err != nil {
		t.Fatalf("MarkRead p2: %v", err)
	}
	if total, _ := chatSvc.UnreadTotal(ctx, p2); total != 0 {
		t.Fatalf("после прочтения у p2 должно быть 0, got=%d", total)
	}
	convs, _ = chatSvc.ListDirect(ctx, p1)
	if convs[0].OtherLastRead != msg.ID {
		t.Fatalf("p1 должен видеть other_last_read=%d, got=%d", msg.ID, convs[0].OtherLastRead)
	}

	t.Log("✅ ЛС: гейт по матчу, комната, доступ, fan-out+уведомление, список, непрочитанные/прочтение")
}
