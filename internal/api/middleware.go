package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const ctxUserID contextKey = "userID"

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(s.cfg.API.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		userID := int64(userIDFloat)

		// Проверяем пользователя через кэш (TTL 60с) — снижает нагрузку на БД при высоком трафике.
		// При бане вызвать globalUserCache.Invalidate(userID) для мгновенного эффекта.
		user, cached := globalUserCache.get(userID)
		if !cached {
			var err error
			user, err = s.userRepo.GetByID(r.Context(), userID)
			if err != nil || user == nil {
				jsonError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			globalUserCache.set(user)
		}
		if user.IsBanned {
			globalUserCache.Invalidate(userID)
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(ctxUserID).(int64)
		if !ok {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Кэш admin-статуса на 5 минут — снижает нагрузку на БД
		if isAdmin, cached := adminStatusCache.get(userID); cached {
			if !isAdmin {
				jsonError(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		isAdmin, err := s.adminRepo.IsAdminByUserID(r.Context(), userID)
		if err != nil {
			jsonError(w, "forbidden", http.StatusForbidden)
			return
		}
		adminStatusCache.set(userID, isAdmin)
		if !isAdmin {
			jsonError(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func currentUserID(r *http.Request) int64 {
	id, _ := r.Context().Value(ctxUserID).(int64)
	return id
}
