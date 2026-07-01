package auth

import (
	"time"

	"mypaas/internal/db"
)

type UserResponse struct {
	ID             string  `json:"id"`
	Email          string  `json:"email"`
	GithubID       *string `json:"githubId"`
	GithubUsername *string `json:"githubUsername"`
	AvatarURL      *string `json:"avatarUrl"`
	Role           string  `json:"role"`
	CreatedAt      string  `json:"createdAt"`
	LastLoginAt    *string `json:"lastLoginAt"`
}

func UserResponseFromDB(user db.User) UserResponse {
	return UserResponse{
		ID:             user.ID.String(),
		Email:          user.Email,
		GithubID:       user.GithubID,
		GithubUsername: user.GithubUsername,
		AvatarURL:      user.AvatarUrl,
		Role:           user.Role,
		CreatedAt:      formatTimestamp(user.CreatedAt.Time, user.CreatedAt.Valid),
		LastLoginAt:    optionalTimestamp(user.LastLoginAt.Time, user.LastLoginAt.Valid),
	}
}

func formatTimestamp(t time.Time, valid bool) string {
	if !valid {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func optionalTimestamp(t time.Time, valid bool) *string {
	if !valid {
		return nil
	}
	formatted := t.UTC().Format(time.RFC3339)
	return &formatted
}
