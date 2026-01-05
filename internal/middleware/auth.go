// internal/middleware/auth.go
package middleware

import (
	"context"
	"net/http"

	"github.com/as-tanais/hofermart/internal/auth"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(jwtManager *auth.JWTManager, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			cookie, err := r.Cookie("token")
			if err != nil {

				log.Debug("No auth cookie found")
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			claims, err := jwtManager.VerifyToken(cookie.Value)
			if err != nil {
				log.Debug("Invalid JWT token", zap.Error(err))

				ClearAuthCookie(w)

				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Немедленное удаление
	})
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

func MustGetUserIDFromContext(ctx context.Context) uuid.UUID {
	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		panic("userID not found in context - middleware not applied")
	}
	return userID
}
