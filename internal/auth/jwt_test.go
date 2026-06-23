package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func writeTestKey(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	path := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}
	return path
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestValidatorSubject(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	keyPath := writeTestKey(t, key)

	v, err := NewValidator(keyPath, "xmine-identity")
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		token := signToken(t, key, jwt.MapClaims{
			"sub": "player-uuid",
			"iss": "xmine-identity",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		sub, err := v.Subject(token)
		if err != nil {
			t.Fatalf("Subject: %v", err)
		}
		if sub != "player-uuid" {
			t.Fatalf("got sub=%q, want player-uuid", sub)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token := signToken(t, key, jwt.MapClaims{
			"sub": "player-uuid",
			"iss": "xmine-identity",
			"exp": time.Now().Add(-time.Hour).Unix(),
		})
		if _, err := v.Subject(token); err != ErrInvalidToken {
			t.Fatalf("got err=%v, want ErrInvalidToken", err)
		}
	})

	t.Run("wrong issuer", func(t *testing.T) {
		token := signToken(t, key, jwt.MapClaims{
			"sub": "player-uuid",
			"iss": "someone-else",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		if _, err := v.Subject(token); err != ErrInvalidToken {
			t.Fatalf("got err=%v, want ErrInvalidToken", err)
		}
	})

	t.Run("wrong signing key", func(t *testing.T) {
		token := signToken(t, otherKey, jwt.MapClaims{
			"sub": "player-uuid",
			"iss": "xmine-identity",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		if _, err := v.Subject(token); err != ErrInvalidToken {
			t.Fatalf("got err=%v, want ErrInvalidToken", err)
		}
	})

	t.Run("missing sub", func(t *testing.T) {
		token := signToken(t, key, jwt.MapClaims{
			"iss": "xmine-identity",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		if _, err := v.Subject(token); err != ErrInvalidToken {
			t.Fatalf("got err=%v, want ErrInvalidToken", err)
		}
	})

	t.Run("garbage token", func(t *testing.T) {
		if _, err := v.Subject("not-a-jwt"); err != ErrInvalidToken {
			t.Fatalf("got err=%v, want ErrInvalidToken", err)
		}
	})
}
