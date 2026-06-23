package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned for any JWT that fails parsing or claim validation.
var ErrInvalidToken = errors.New("invalid token")

// Validator performs offline RS256 validation of JWTs issued by the Identity Service.
type Validator struct {
	publicKey *rsa.PublicKey
	issuer    string
}

// NewValidator loads the Identity Service's RSA public key from a PEM file.
func NewValidator(publicKeyPath, issuer string) (*Validator, error) {
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read jwt public key: %w", err)
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse jwt public key: %w", err)
	}
	return &Validator{publicKey: key, issuer: issuer}, nil
}

// Subject validates the token (signature, alg, iss, exp) and returns its "sub" claim
// (the player uuid), or ErrInvalidToken if validation fails.
func (v *Validator) Subject(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != "RS256" {
			return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
		}
		return v.publicKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(v.issuer))
	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidToken
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", ErrInvalidToken
	}
	return sub, nil
}
