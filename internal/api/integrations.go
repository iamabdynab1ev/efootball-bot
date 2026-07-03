package api

import (
	"encoding/json"
	"net/http"

	"efootball-bot/internal/groupcast"
	"efootball-bot/internal/wa"
)

// SetTGGroup подключает канал «Telegram-группа» (для статуса в админке).
func (s *Server) SetTGGroup(t *groupcast.TelegramGroup) { s.tgGroup = t }

// SetWhatsApp подключает канал WhatsApp (nil — выключен).
func (s *Server) SetWhatsApp(c *wa.Client) { s.waClient = c }

// handleIntegrations — GET /api/admin/integrations: статусы групповых каналов.
func (s *Server) handleIntegrations(w http.ResponseWriter, r *http.Request) {
	tg := map[string]any{"connected": false}
	if s.tgGroup != nil && s.tgGroup.ChatID() != 0 {
		tg["connected"] = true
		tg["chat_id"] = s.tgGroup.ChatID()
	}
	waState := map[string]any{"enabled": false}
	if s.waClient != nil {
		status, group := s.waClient.Status()
		waState["enabled"] = true
		waState["status"] = status
		waState["group_jid"] = group
	}
	jsonOK(w, map[string]any{"telegram": tg, "whatsapp": waState})
}

// handleWAQR — GET /api/admin/wa/qr: PNG с QR-кодом для входа в WhatsApp.
func (s *Server) handleWAQR(w http.ResponseWriter, r *http.Request) {
	if s.waClient == nil {
		jsonError(w, "WhatsApp выключен (WA_ENABLED)", http.StatusNotFound)
		return
	}
	png := s.waClient.QRPNG()
	if len(png) == 0 {
		jsonError(w, "QR сейчас не нужен (уже подключено или идёт соединение)", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

// handleWAGroups — GET /api/admin/wa/groups: группы подключённого аккаунта.
func (s *Server) handleWAGroups(w http.ResponseWriter, r *http.Request) {
	if s.waClient == nil {
		jsonError(w, "WhatsApp выключен", http.StatusNotFound)
		return
	}
	groups, err := s.waClient.Groups(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	jsonOK(w, map[string]any{"groups": groups})
}

// handleWASetGroup — POST /api/admin/wa/group {jid}: выбрать группу ("" — отключить).
func (s *Server) handleWASetGroup(w http.ResponseWriter, r *http.Request) {
	if s.waClient == nil {
		jsonError(w, "WhatsApp выключен", http.StatusNotFound)
		return
	}
	var req struct {
		JID string `json:"jid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "неверный запрос", http.StatusBadRequest)
		return
	}
	if err := s.waClient.SetGroup(r.Context(), req.JID); err != nil {
		jsonError(w, "неверный JID группы", http.StatusBadRequest)
		return
	}
	jsonOK(w, map[string]any{"ok": true, "group_jid": req.JID})
}
