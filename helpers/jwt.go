package helpers

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// tokenTTL is how long an issued token stays valid.
const tokenTTL = 24 * time.Hour

// errNoJWTSecret is returned when JWT_SECRET is not configured. The app must
// never fall back to a well-known development secret: a predictable signing key
// lets anyone forge valid tokens for any user.
var errNoJWTSecret = errors.New("JWT_SECRET is not set; refusing to sign or verify tokens with a default secret")

// jwtSecret returns the signing secret from the environment, or an error when
// it is unset. Fails closed in every environment — there is no fallback secret.
func jwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errNoJWTSecret
	}
	return []byte(secret), nil
}

// Claims is the set of data embedded in the JWT.
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given user ID.
func GenerateToken(userID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	return token.SignedString(secret)
}

// ParseToken validates a token string and returns its claims.
func ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		// Ensure the signing method is the one we expect (HMAC).
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret()
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
