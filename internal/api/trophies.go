package api

import (
	"efootball-bot/internal/service"
	"net/http"
)

// trophyItem — одна неполученная награда для церемонии «Забрать».
// Клиент рисует имя через каталог trophyCat по Key, иконку берёт отсюда.
type trophyItem struct {
	Kind    string `json:"kind"`    // "trophy" | "achievement"
	Key     string `json:"key"`     // award_type трофея или код достижения
	Icon    string `json:"icon"`    // эмодзи награды
	Name    string `json:"name"`    // русский фолбэк-заголовок
	Context string `json:"context"` // название лиги (для трофеев) либо ""
}

// unclaimedTrophies — все неполученные награды игрока (трофеи турниров +
// достижения) единым списком, старые первыми: церемония «Забрать все» идёт
// по хронологии получения.
func (s *Server) unclaimedTrophies(r *http.Request, userID int64) []trophyItem {
	items := []trophyItem{}
	if s.awardRepo != nil {
		if aw, err := s.awardRepo.GetUnclaimedByUser(r.Context(), userID); err == nil {
			for _, a := range aw {
				items = append(items, trophyItem{
					Kind:    "trophy",
					Key:     a.AwardType,
					Icon:    service.AwardEmoji(a.AwardType),
					Name:    service.AwardLabel(a.AwardType),
					Context: a.LeagueName,
				})
			}
		}
	}
	if s.achievRepo != nil {
		if ach, err := s.achievRepo.GetUnclaimed(r.Context(), userID); err == nil {
			for _, ua := range ach {
				icon, name, key := "🏅", "", ""
				if ua.Achievement != nil {
					icon, name, key = ua.Achievement.Icon, ua.Achievement.NameRu, ua.Achievement.Code
				}
				items = append(items, trophyItem{Kind: "achievement", Key: key, Icon: icon, Name: name})
			}
		}
	}
	return items
}

// handleUnclaimedTrophies — GET /api/me/trophies/unclaimed — список наград,
// ожидающих получения в Трофейной комнате (кнопка «Забрать»).
func (s *Server) handleUnclaimedTrophies(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]any{"items": s.unclaimedTrophies(r, currentUserID(r))})
}

// handleClaimTrophies — POST /api/me/trophies/claim — помечает все награды
// игрока полученными и возвращает их список для церемонии по одному.
func (s *Server) handleClaimTrophies(w http.ResponseWriter, r *http.Request) {
	uid := currentUserID(r)
	// Снимок ДО пометки — его и празднуем; пометка идемпотентна.
	items := s.unclaimedTrophies(r, uid)
	if s.awardRepo != nil {
		if _, err := s.awardRepo.ClaimAll(r.Context(), uid); err != nil {
			jsonError(w, "db error", http.StatusInternalServerError)
			return
		}
	}
	if s.achievRepo != nil {
		if _, err := s.achievRepo.ClaimAll(r.Context(), uid); err != nil {
			jsonError(w, "db error", http.StatusInternalServerError)
			return
		}
	}
	jsonOK(w, map[string]any{"items": items})
}
