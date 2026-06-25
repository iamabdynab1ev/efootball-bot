package api

import (
	"io"
	"net/http"
	"time"

	"efootball-bot/internal/data"
)

var logoProxyClient = &http.Client{Timeout: 6 * time.Second}

// handleClubLogo — лёгкий CORS-прокси гербов клубов для отрисовки карточки в
// браузере (canvas). Только СТРИМИТ байты (io.Copy) — без декодирования и без
// удержания в памяти, поэтому безопасно для маленького инстанса.
// CDN TheSportsDB не отдаёт CORS, а без него canvas нельзя экспортировать.
func (s *Server) handleClubLogo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("club")
	url := data.ClubLogos[id]
	if url == "" {
		http.NotFound(w, r)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		http.Error(w, "bad url", http.StatusBadGateway)
		return
	}
	resp, err := logoProxyClient.Do(req)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "logo not found", http.StatusBadGateway)
		return
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/png"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=604800") // 7 дней
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 3<<20))
}
