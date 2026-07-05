package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"mypaas/internal/db"
)

const (
	AccessCookieName  = "mypaas_access"
	RefreshCookieName = "mypaas_refresh"
)

type Claims struct {
	UserID   uuid.UUID `json:"userId"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	TokenUse string    `json:"tokenUse"`
	jwt.RegisteredClaims
}

type Tokens struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
}

type TokenService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenService(secret string) (*TokenService, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 bytes")
	}
	return &TokenService{
		secret:     []byte(secret),
		accessTTL:  24 * time.Hour,
		refreshTTL: 30 * 24 * time.Hour,
	}, nil
}

func (s *TokenService) Issue(user db.User) (Tokens, error) {
	accessToken, accessExpiresAt, err := s.issue(user, "access", s.accessTTL)
	if err != nil {
		return Tokens{}, err
	}
	refreshToken, refreshExpiresAt, err := s.issue(user, "refresh", s.refreshTTL)
	if err != nil {
		return Tokens{}, err
	}

	return Tokens{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresAt:        accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *TokenService) issue(user db.User, tokenUse string, ttl time.Duration) (string, time.Time, error) {
	expiresAt := time.Now().Add(ttl)
	claims := Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Role:     user.Role,
		TokenUse: tokenUse,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}

	return signed, expiresAt, nil
}

func (s *TokenService) Parse(raw string) (Claims, error) {
	return s.ParseAccess(raw)
}

func (s *TokenService) ParseAccess(raw string) (Claims, error) {
	claims, err := s.parse(raw)
	if err != nil {
		return Claims{}, err
	}
	if claims.TokenUse != "" && claims.TokenUse != "access" {
		return Claims{}, errors.New("jwt is not an access token")
	}
	return claims, nil
}

func (s *TokenService) ParseRefresh(raw string) (Claims, error) {
	claims, err := s.parse(raw)
	if err != nil {
		return Claims{}, err
	}
	if claims.TokenUse != "refresh" {
		return Claims{}, errors.New("jwt is not a refresh token")
	}
	return claims, nil
}

func (s *TokenService) parse(raw string) (Claims, error) {
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
