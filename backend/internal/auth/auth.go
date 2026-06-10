package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/morphatrix/campmenu/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash at the given cost.
func HashPassword(plain string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	return string(b), err
}

// CheckPassword reports whether plain matches the stored hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// RandomToken returns a URL-safe random hex token of n bytes.
func RandomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Claims is the JWT payload. Imp, when set, is the admin currently impersonating
// this user (so the session can switch back).
type Claims struct {
	UserID uuid.UUID   `json:"uid"`
	Role   models.Role `json:"role"`
	Imp    uuid.UUID   `json:"imp,omitempty"`
	jwt.RegisteredClaims
}

// GenerateJWT signs a token for a user.
func GenerateJWT(secret string, expiry time.Duration, userID uuid.UUID, role models.Role) (string, error) {
	return GenerateImpersonationJWT(secret, expiry, userID, role, uuid.Nil)
}

// GenerateImpersonationJWT signs a token for userID, recording the impersonating
// admin (imp). Pass uuid.Nil for a normal session.
func GenerateImpersonationJWT(secret string, expiry time.Duration, userID uuid.UUID, role models.Role, imp uuid.UUID) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		Imp:    imp,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID.String(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseJWT validates a token and returns its claims.
func ParseJWT(secret, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
