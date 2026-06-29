package middleware

import (
	"context"
	"net/http"
	"strings"
	"xmine/litebans-api/api"
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
// player uuid (JWT "sub") into the request context. This is the authentication
// step: it establishes who the caller is, but makes no decision about what
// they're allowed to do.
func (a *Auth) RequireJWT() AuthStep {
	return func(next api.StrictHandlerFunc) api.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			token, ok := bearerToken(r)
			if !ok {
				return nil, domain.NewUnauthorized("missing bearer token")
			}
			uuid, err := a.validator.Subject(token)
			if err != nil {
				return nil, domain.NewUnauthorized("invalid or expired token")
			}
			ctx = context.WithValue(ctx, playerUUIDKey, uuid)
			return next(ctx, w, r, request)
		}
	}
}

// RequireModPermission is the authorization step: it requires that the caller
// has already been authenticated earlier in the chain (e.g. by RequireJWT,
// which places the player uuid into the context) and checks that the
// authenticated player holds the configured moderator permission. It does not
// parse or validate JWTs itself — if no player uuid is present in the context,
// that's a chain-composition error (this step was used without an
// authentication step ahead of it), and the request is rejected as unauthorized.
func (a *Auth) RequireModPermission() AuthStep {
	return func(next api.StrictHandlerFunc) api.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			uuid, ok := PlayerUUIDFromContext(ctx)
			if !ok {
				return nil, domain.NewUnauthorized("missing authenticated player")
			}
			hasPermission, err := a.authority.HasPermission(ctx, uuid, a.modPermission)
			if err != nil {
				return nil, domain.NewServiceUnavailable("failed to verify permissions", err)
			}
			if !hasPermission {
				return nil, domain.NewForbidden("missing required permission")
			}
			return next(ctx, w, r, request)
		}
	}
}

// OptionalJWT parses a JWT if present and valid, placing the player uuid into the
// request context, but never rejects the request if the token is absent/invalid.
func (a *Auth) OptionalJWT() AuthStep {
	return func(next api.StrictHandlerFunc) api.StrictHandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
			token, ok := bearerToken(r)
			if !ok {
				return next(ctx, w, r, request)
			}
			uuid, err := a.validator.Subject(token)
			if err != nil {
				return next(ctx, w, r, request)
			}
			ctx = context.WithValue(ctx, playerUUIDKey, uuid)
			return next(ctx, w, r, request)
		}
	}
}
