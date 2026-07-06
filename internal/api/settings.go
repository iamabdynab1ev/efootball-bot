package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Ключи настроек поддержки.
const (
	keySupportPhone    = "support_phone"
	keySupportWhatsapp = "support_whatsapp"
	keySupportTelegram = "support_telegram"
)

var supportKeys = []string{keySupportPhone, keySupportWhatsapp, keySupportTelegram}

type supportContact struct {
	Phone    string `json:"phone"`
	Whatsapp string `json:"whatsapp"`
	Telegram string `json:"telegram"`
}

// handleGetSupport — публично отдаёт контакты поддержки.
func (s *Server) handleGetSupport(w http.ResponseWriter, r *http.Request) {
	if s.settingsRepo == nil {
		jsonOK(w, supportContact{})
		return
	}
	m, err := s.settingsRepo.GetMany(r.Context(), supportKeys)
	if err != nil {
		jsonOK(w, supportContact{})
		return
	}
	jsonOK(w, supportContact{
		Phone:    m[keySupportPhone],
		Whatsapp: m[keySupportWhatsapp],
		Telegram: m[keySupportTelegram],
	})
}

// handleSetSupport — админ задаёт контакты поддержки.
func (s *Server) handleSetSupport(w http.ResponseWriter, r *http.Request) {
	if s.settingsRepo == nil {
		jsonError(w, "settings not configured", http.StatusServiceUnavailable)
		return
	}
	var body supportContact
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	vals := map[string]string{
		keySupportPhone:    strings.TrimSpace(body.Phone),
		keySupportWhatsapp: strings.TrimSpace(body.Whatsapp),
		keySupportTelegram: strings.TrimSpace(body.Telegram),
	}
	for k, v := range vals {
		if len(v) > 100 {
			jsonError(w, "value too long", http.StatusBadRequest)
			return
		}
		if err := s.settingsRepo.Set(r.Context(), k, v); err != nil {
			jsonError(w, "failed to save", http.StatusInternalServerError)
			return
		}
	}
	jsonOK(w, body)
}
