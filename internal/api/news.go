package api

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"efootball-bot/internal/groupcast"
	"efootball-bot/internal/models"
)

// Новости турнира в общую группу (WhatsApp/Telegram): каждое заметное
// событие — читаемое сообщение с *жирными* заголовками (WhatsApp-разметка),
// эмодзи и короткими строками — легко читается на телефоне.

// SetGroupHub подключает шину групповых новостей.
func (s *Server) SetGroupHub(h *groupcast.Hub) { s.groupHub = h }

func (s *Server) publishNews(text string) {
	if s.groupHub == nil || text == "" {
		return
	}
	s.groupHub.Publish(text)
}

// newsLeagueOpen — «набор открыт»: название, формат, дедлайн набора.
func (s *Server) newsLeagueOpen(league *models.League) {
	if league == nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "📣 *НОВАЯ ЛИГА: НАБОР ОТКРЫТ!*\n\n")
	fmt.Fprintf(&b, "🏆 *%s*\n", league.Name)
	if f := formatLabelRu(league.RoundsType); f != "" {
		fmt.Fprintf(&b, "⚙️ Формат: %s\n", f)
	}
	if league.RegistrationDeadline != nil {
		fmt.Fprintf(&b, "🗓 Заявки до: *%s*\n", league.RegistrationDeadline.In(dushanbeTZ()).Format("02.01 15:04"))
	}
	b.WriteString("\nПодавайте заявку в приложении — место в таблице ждёт! ⚽")
	s.publishNews(b.String())
}

// announceLeagueRegistration — реклама нового турнира ЛИЧНО каждому игроку:
// push (всем подписанным) + Telegram (всем привязанным) + колокольчик (всем).
// Групповая новость уходит отдельно через newsLeagueOpen. Запускать в горутине.
func (s *Server) announceLeagueRegistration(ctx context.Context, league *models.League) {
	if league == nil {
		return
	}
	body := "«" + league.Name + "»"
	if f := formatLabelRu(league.RoundsType); f != "" {
		body += " · " + f
	}
	if league.RegistrationDeadline != nil {
		body += " · до " + league.RegistrationDeadline.In(dushanbeTZ()).Format("02.01 15:04")
	}
	body += ". Успей подать заявку — место в таблице ждёт! ⚽"

	if s.webPush != nil {
		s.webPush.Broadcast("🏆 Новый турнир открыт!", body, "/leagues")
	}
	if s.notifier != nil {
		var b strings.Builder
		b.WriteString("🏆🔥 *НОВЫЙ ТУРНИР — НАБОР ОТКРЫТ!*\n\n")
		fmt.Fprintf(&b, "⚽ *%s*\n", league.Name)
		if f := formatLabelRu(league.RoundsType); f != "" {
			fmt.Fprintf(&b, "⚙️ Формат: %s\n", f)
		}
		if league.RegistrationDeadline != nil {
			fmt.Fprintf(&b, "🗓 Заявки до: *%s*\n", league.RegistrationDeadline.In(dushanbeTZ()).Format("02.01 15:04"))
		}
		b.WriteString("\nЗаходи в приложение и подавай заявку! 🚀")
		if ids, err := s.userRepo.GetAllTelegramIDs(ctx); err == nil && len(ids) > 0 {
			s.notifier.BroadcastCustom(b.String(), ids)
		}
	}
	if ids, err := s.userRepo.GetAllUserIDs(ctx); err == nil && len(ids) > 0 {
		s.notifyT(ctx, ids, models.NotifTournament, "/leagues", func(string) (string, string) {
			return "🏆 Новый турнир!", body
		})
	}
}

// newsDrawDone — жеребьёвка: составы групп.
func (s *Server) newsDrawDone(ctx context.Context, leagueID int64) {
	league, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil || league == nil {
		return
	}
	members, err := s.leagueRepo.GetMembers(ctx, leagueID)
	if err != nil {
		return
	}
	byGroup := map[string][]string{}
	for _, m := range members {
		if m.Status != models.MemberApproved {
			continue
		}
		name := "?"
		if u, uErr := s.userRepo.GetByID(ctx, m.UserID); uErr == nil && u != nil {
			name = u.DisplayName
		}
		byGroup[m.GroupName] = append(byGroup[m.GroupName], name)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "🎲 *ЖЕРЕБЬЁВКА ПРОВЕДЕНА!*\n\n🏆 *%s*\n", league.Name)
	groups := make([]string, 0, len(byGroup))
	for g := range byGroup {
		groups = append(groups, g)
	}
	sort.Strings(groups)
	for _, g := range groups {
		if g == "" {
			continue
		}
		fmt.Fprintf(&b, "\n*Группа %s:*\n", g)
		for _, n := range byGroup[g] {
			fmt.Fprintf(&b, "  • %s\n", n)
		}
	}
	if noGroup, ok := byGroup[""]; ok && len(groups) <= 1 {
		b.WriteString("\n*Участники:*\n")
		for _, n := range noGroup {
			fmt.Fprintf(&b, "  • %s\n", n)
		}
	}
	b.WriteString("\nРасписание уже в приложении — удачи! ⚽")
	s.publishNews(b.String())
}

// newsPlayoffDraw — пары плей-офф после жеребьёвки.
func (s *Server) newsPlayoffDraw(ctx context.Context, leagueID int64) {
	league, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil || league == nil {
		return
	}
	slots, err := s.bracketRepo.GetAllSlots(ctx, leagueID)
	if err != nil {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "⚔️ *ПЛЕЙ-ОФФ: ПАРЫ ОПРЕДЕЛЕНЫ!*\n\n🏆 *%s*\n", league.Name)
	currentStage := ""
	for _, sl := range slots {
		if sl.HomeName == "" && sl.AwayName == "" {
			continue
		}
		if sl.Stage != currentStage {
			currentStage = sl.Stage
			label := models.StageLabel[sl.Stage]
			if label == "" {
				label = sl.Stage
			}
			fmt.Fprintf(&b, "\n*%s:*\n", label)
		}
		home, away := sl.HomeName, sl.AwayName
		if home == "" {
			home = "—"
		}
		if away == "" {
			away = "—"
		}
		fmt.Fprintf(&b, "  ⚽ %s  🆚  %s\n", home, away)
	}
	b.WriteString("\nКаждый матч — на вылет. Вперёд! 🔥")
	s.publishNews(b.String())
}

// newsMatchResult — подтверждённый результат: короткая читаемая строка.
func (s *Server) newsMatchResult(ctx context.Context, m *models.Match, homeName, awayName string) {
	if m == nil || m.HomeGoals == nil || m.AwayGoals == nil {
		return
	}
	league, err := s.leagueRepo.GetByID(ctx, m.LeagueID)
	if err != nil || league == nil {
		return
	}
	stage := ""
	if models.IsKnockoutStage(m.Stage) {
		if l := models.StageLabel[m.Stage]; l != "" {
			stage = " · " + l
		}
	} else {
		stage = fmt.Sprintf(" · Тур %d", m.Round)
	}
	emoji := "⚽"
	if *m.HomeGoals == *m.AwayGoals {
		emoji = "🤝"
	}
	s.publishNews(fmt.Sprintf("%s *%s %d:%d %s*\n🏆 %s%s",
		emoji, homeName, *m.HomeGoals, *m.AwayGoals, awayName, league.Name, stage))
}

// NewsChampion — чемпион турнира: финальный залп в группу.
// Вызывается сервисом матчей при закрытии турнира.
func (s *Server) NewsChampion(ctx context.Context, leagueID, championID int64) {
	league, err := s.leagueRepo.GetByID(ctx, leagueID)
	if err != nil || league == nil {
		return
	}
	champ, err := s.userRepo.GetByID(ctx, championID)
	if err != nil || champ == nil {
		return
	}
	var b strings.Builder
	b.WriteString("🏆🏆🏆 *У НАС ЧЕМПИОН!* 🏆🏆🏆\n\n")
	fmt.Fprintf(&b, "👑 *%s* — победитель турнира\n", champ.DisplayName)
	fmt.Fprintf(&b, "🏟 *%s*\n", league.Name)
	fmt.Fprintf(&b, "\n📅 %s\n", time.Now().In(dushanbeTZ()).Format("02.01.2006"))
	b.WriteString("\nПоздравляем! Все награды — в приложении 🎉")
	s.publishNews(b.String())
}

// formatLabelRu — русское имя формата лиги для новостей.
func formatLabelRu(roundsType string) string {
	switch roundsType {
	case "hybrid", "groups", "groups_playoff":
		return "группы + плей-офф"
	case "single":
		return "один круг"
	case "double":
		return "два круга"
	case "cup":
		return "кубок (на вылет)"
	case "swiss":
		return "швейцарская система"
	case "double_elim":
		return "двойная элиминация"
	}
	return roundsType
}
