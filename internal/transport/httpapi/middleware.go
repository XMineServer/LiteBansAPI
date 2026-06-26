package httpapi

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

// StrictMiddleware dispatches per-operation auth requirements by operationId, since the
// generated StrictServerInterface has a single handler signature for all operations.
func (a *Auth) StrictMiddleware(f api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
	switch operationID {
	case "GetPlayerPunishmentsMe":
		return a.requireJWT(f)
	case "GetModPunishmentsList":
		return a.requireModPermission(f)
	case "GetPunishmentByID":
		return a.optionalJWT(f)
	default:
		return f
	}
}

// requireJWT rejects requests without a valid JWT (401), otherwise places the
// player uuid (JWT "sub") into the request context.
func (a *Auth) requireJWT(next api.StrictHandlerFunc) api.StrictHandlerFunc {
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

// requireModPermission requires a valid JWT and the configured moderator permission,
// returning 401/403/503 as appropriate.
func (a *Auth) requireModPermission(next api.StrictHandlerFunc) api.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		token, ok := bearerToken(r)
		if !ok {
			return nil, domain.NewUnauthorized("missing bearer token")
		}
		uuid, err := a.validator.Subject(token)
		if err != nil {
			return nil, domain.NewUnauthorized("invalid or expired token")
		}
		hasPermission, err := a.authority.HasPermission(ctx, uuid, a.modPermission)
		if err != nil {
			return nil, domain.NewServiceUnavailable("failed to verify permissions", err)
		}
		if !hasPermission {
			return nil, domain.NewForbidden("missing required permission")
		}
		ctx = context.WithValue(ctx, playerUUIDKey, uuid)
		return next(ctx, w, r, request)
	}
}

// optionalJWT parses a JWT if present and valid, placing the player uuid into the
// request context, but never rejects the request if the token is absent/invalid.
func (a *Auth) optionalJWT(next api.StrictHandlerFunc) api.StrictHandlerFunc {
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
