package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT payload used across services.
type Claims struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// TokenService handles JWT creation and validation.
type TokenService struct {
	secret    []byte
	expiresIn time.Duration
}

// NewTokenService returns configured token service.
func NewTokenService(secret string, expiresIn time.Duration) *TokenService {
	if expiresIn <= 0 {
		expiresIn = time.Hour
	}
	return &TokenService{secret: []byte(secret), expiresIn: expiresIn}
}

// GenerateToken issues JWT for given user.
func (t *TokenService) GenerateToken(userID int64, role string) (string, error) {
	if userID == 0 {
		return "", errors.New("token: user id is required")
	}

	now := time.Now().UTC()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.expiresIn)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.secret)
}

// ValidateToken verifies and decodes JWT.
func (t *TokenService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("token: unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("token: invalid claims")
}

