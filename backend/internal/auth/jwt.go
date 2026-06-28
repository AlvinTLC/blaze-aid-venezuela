// Package auth issues and verifies the session JWTs minted after a successful
// magic-login verification.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionTTL is how long an issued session token stays valid.
const SessionTTL = 24 * time.Hour

const issuer = "blazeaid-hub"

// IssueJWT mints an HS256 session token for subject, valid for SessionTTL.
func IssueJWT(secret, subject string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(SessionTTL)
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// ParseJWT validates a token's signature and expiry, returning its subject.
func ParseJWT(secret, tokenString string) (string, error) {
	var claims jwt.RegisteredClaims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", err
	}
	if !token.Valid {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}
