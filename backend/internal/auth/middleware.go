package auth

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"mypaas/internal/db"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
)

func Middleware(tokens *TokenService, queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				if cookie, err := r.Cookie(AccessCookieName); err == nil {
					raw = cookie.Value
				}
			}
			if raw == "" {
				httpx.DomainError(w, errs.ErrUnauthorized)
				return
			}

			claims, err := tokens.Parse(raw)
			if err != nil {
				httpx.DomainError(w, errs.ErrUnauthorized)
				return
			}

			user, err := queries.GetUserByID(r.Context(), claims.UserID)
			if err != nil {
				if err == pgx.ErrNoRows {
					httpx.DomainError(w, errs.ErrUnauthorized)
					return
				}
				httpx.DomainError(w, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), User{
				ID:    user.ID,
				Email: user.Email,
				Role:  user.Role,
			})))
		})
	}
}

func RequireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := CurrentUser(r)
		if err != nil {
			httpx.DomainError(w, err)
			return
		}
		if user.Role != "owner" {
			httpx.DomainError(w, errs.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	value, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
