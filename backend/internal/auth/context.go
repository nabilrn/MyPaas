package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"mypaas/internal/errs"
)

type contextKey string

const userContextKey contextKey = "authUser"

type User struct {
	ID    uuid.UUID
	Email string
	Role  string
}

func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func CurrentUser(r *http.Request) (User, error) {
	user, ok := r.Context().Value(userContextKey).(User)
	if !ok {
		return User{}, errs.ErrUnauthorized
	}
	return user, nil
}
