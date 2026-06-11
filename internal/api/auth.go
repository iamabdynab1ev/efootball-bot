package api

import (
	"context"
	"efootball-bot/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type googleAuthRequest struct {
	IDToken string `json:"id_token"`
}

type googleTokenInfo struct {
	Sub       string `json:"sub"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Aud       string `json:"aud"`
	Error     string `json:"error"`
	ErrorDesc string `json:"error_description"`
}

func (s *Server) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	var req googleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	info, err := verifyGoogleToken(req.IDToken, s.cfg.API.GoogleClientID)
	if err != nil || info.Sub == "" {
		jsonError(w, "invalid google token", http.StatusUnauthorized)
		return
	}

	user, err := s.userRepo.UpsertByGoogle(r.Context(), info.Sub, info.Email, info.Name)
	if err != nil || user == nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(s.cfg.API.JWTSecret))
	if err != nil {
		jsonError(w, "token error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"token": signed,
		"user":  s.userDTOWithRole(r.Context(), user),
	})
}

// handleAdminLogin — вход по логину и паролю для администраторов.
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	cred, err := s.adminRepo.GetAdminCredential(r.Context(), req.Username)
	if err != nil || cred == nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(req.Password)); err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	user, err := s.userRepo.GetByID(r.Context(), cred.UserID)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusInternalServerError)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(s.cfg.API.JWTSecret))
	if err != nil {
		jsonError(w, "token error", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"token": signed,
		"user":  s.userDTOWithRole(r.Context(), user),
	})
}

// handleDevLogin — быстрый вход по user_id, только в development режиме.
func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Env == "production" {
		jsonError(w, "not available", http.StatusNotFound)
		return
	}
	var req struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == 0 {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	user, err := s.userRepo.GetByID(r.Context(), req.UserID)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(s.cfg.API.JWTSecret))
	if err != nil {
		jsonError(w, "token error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{
		"token": signed,
		"user":  s.userDTOWithRole(r.Context(), user),
	})
}

// handleDevUsers — список всех пользователей для dev-логина.
func (s *Server) handleDevUsers(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Env == "production" {
		jsonError(w, "not available", http.StatusNotFound)
		return
	}
	users, err := s.userRepo.GetAllByRating(r.Context(), 100)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	result := make([]map[string]any, 0, len(users))
	for _, u := range users {
		result = append(result, map[string]any{
			"id":           u.ID,
			"display_name": u.DisplayName,
			"rating":       u.Rating,
			"rank":         u.Rank,
		})
	}
	jsonOK(w, result)
}

func verifyGoogleToken(idToken, clientID string) (*googleTokenInfo, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf(
		"https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var info googleTokenInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	if info.Error != "" {
		return nil, fmt.Errorf("google: %s", info.Error)
	}
	if clientID != "" && info.Aud != clientID {
		return nil, fmt.Errorf("google token audience mismatch")
	}
	return &info, nil
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, err := s.userRepo.GetByID(r.Context(), currentUserID(r))
	if err != nil || user == nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, s.userDTOWithRole(r.Context(), user))
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DisplayName  string  `json:"display_name"`
		TeamPower    int     `json:"team_power"`
		FavoriteClub *string `json:"favorite_club"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	uid := currentUserID(r)
	if body.DisplayName != "" {
		body.DisplayName = strings.TrimSpace(body.DisplayName)
		if len(body.DisplayName) < 1 || len(body.DisplayName) > 50 {
			jsonError(w, "display_name must be 1-50 characters", http.StatusBadRequest)
			return
		}
		if err := s.userRepo.UpdateDisplayName(r.Context(), uid, body.DisplayName); err != nil {
			jsonError(w, "failed to update name", http.StatusInternalServerError)
			return
		}
	}
	if body.TeamPower > 0 && body.TeamPower <= 9999999 {
		if err := s.userRepo.UpdateTeamPower(r.Context(), uid, body.TeamPower); err != nil {
			jsonError(w, "failed to update team power", http.StatusInternalServerError)
			return
		}
	}
	if body.FavoriteClub != nil {
		if err := s.userRepo.UpdateFavoriteClub(r.Context(), uid, *body.FavoriteClub); err != nil {
			jsonError(w, "failed to update favorite club", http.StatusInternalServerError)
			return
		}
	}
	user, err := s.userRepo.GetByID(r.Context(), uid)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	jsonOK(w, s.userDTOWithRole(r.Context(), user))
}

func (s *Server) handleGenerateLinkCode(w http.ResponseWriter, r *http.Request) {
	code, err := s.userRepo.GenerateLinkCode(r.Context(), currentUserID(r))
	if err != nil {
		jsonError(w, "error generating code", http.StatusInternalServerError)
		return
	}
	resp := map[string]string{
		"code":       code,
		"expires_in": "10 минут",
	}
	if s.cfg.Telegram.BotUsername != "" {
		resp["deep_link"] = "https://t.me/" + s.cfg.Telegram.BotUsername + "?start=link_" + code
	}
	jsonOK(w, resp)
}

// userDTO — маппинг User → JSON-объект.
func userDTO(u *models.User) map[string]any {
	m := map[string]any{
		"id":            u.ID,
		"display_name":  u.DisplayName,
		"rating":        u.Rating,
		"rank":          u.Rank,
		"team_power":    u.TeamPower,
		"language":      u.Language,
		"has_telegram":  u.HasTelegram(),
		"favorite_club": "",
	}
	if u.FavoriteClub != nil {
		m["favorite_club"] = *u.FavoriteClub
	}
	if u.Email != nil {
		m["email"] = *u.Email
	}
	if u.Username != nil {
		m["username"] = *u.Username
	}
	return m
}

func (s *Server) userDTOWithRole(ctx context.Context, u *models.User) map[string]any {
	m := userDTO(u)
	role, _ := s.adminRepo.GetAdminRoleByUserID(ctx, u.ID)
	m["is_admin"] = role != ""
	m["is_super_admin"] = role == "super_admin"
	return m
}
