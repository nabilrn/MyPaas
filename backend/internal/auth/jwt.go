package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"mypaas/internal/db"
)

const AccessCookieName = "mypaas_access"

type Claims struct {
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type Tokens struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenService(secret string) (*TokenService, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes")
	}
	return &TokenService{
		secret: []byte(secret),
		ttl:    24 * time.Hour,
	}, nil
}

func (s *TokenService) Issue(user db.User) (Tokens, error) {
	expiresAt := time.Now().Add(s.ttl)
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return Tokens{}, fmt.Errorf("sign jwt: %w", err)
	}

	return Tokens{AccessToken: signed, ExpiresAt: expiresAt}, nil
}

func (s *TokenService) Parse(raw string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %s", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return Claims{}, fmt.Errorf("parse jwt: %w", err)
	}
	if !token.Valid {
		return Claims{}, errors.New("invalid jwt")
	}
	return claims, nil
}
