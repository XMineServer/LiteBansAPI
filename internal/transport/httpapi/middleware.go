package httpapi

import (
	"context"
	"net/http"
	"strings"
	"xmine/litebans-api/internal/auth"
	"xmine/litebans-api/internal/domain"
)

type contextKey int

const playerUUIDKey contextKey = iota

// PlayerUUIDFromContext returns the player uuid placed in the request context by
// RequireJWT/RequireModPermission/OptionalJWT, if a valid JWT was present.
func PlayerUUIDFromContext(ctx context.Context) (string, bool) {
	uuid, ok := ctx.Value(playerUUIDKey).(string)
	return uuid, ok
}

// Auth holds the components needed to authenticate/authorize incoming requests.
type Auth struct {
	validator     *auth.Validator
	authority     *auth.AuthorityClient
	modPermission string
}

func NewAuth(validator *auth.Validator, authority *auth.AuthorityClient, modPermission string) *Auth {
	return &Auth{validator: validator, authority: authority, modPermission: modPermission}
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

// RequireJWT rejects requests without a valid JWT (401), otherwise places the
// player uuid (JWT "sub") into the request context.
func (a *Auth) RequireJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, domain.NewUnauthorized("missing bearer token"))
			return
		}
		uuid, err := a.validator.Subject(token)
		if err != nil {
			writeError(w, domain.NewUnauthorized("invalid or expired token"))
			return
		}
		ctx := context.WithValue(r.Context(), playerUUIDKey, uuid)
		next(w, r.WithContext(ctx))
	}
}

// RequireModPermission requires a valid JWT and the configured moderator permission,
// returning 401/403/503 as appropriate.
func (a *Auth) RequireModPermission(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeError(w, domain.NewUnauthorized("missing bearer token"))
			return
		}
		uuid, err := a.validator.Subject(token)
		if err != nil {
			writeError(w, domain.NewUnauthorized("invalid or expired token"))
			return
		}
		hasPermission, err := a.authority.HasPermission(r.Context(), uuid, a.modPermission)
		if err != nil {
			writeError(w, domain.NewServiceUnavailable("failed to verify permissions", err))
			return
		}
		if !hasPermission {
			writeError(w, domain.NewForbidden("missing required permission"))
			return
		}
		ctx := context.WithValue(r.Context(), playerUUIDKey, uuid)
		next(w, r.WithContext(ctx))
	}
}

// OptionalJWT parses a JWT if present and valid, placing the player uuid into the
// request context, but never rejects the request if the token is absent/invalid.
func (a *Auth) OptionalJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			next(w, r)
			return
		}
		uuid, err := a.validator.Subject(token)
		if err != nil {
			next(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), playerUUIDKey, uuid)
		next(w, r.WithContext(ctx))
	}
}
