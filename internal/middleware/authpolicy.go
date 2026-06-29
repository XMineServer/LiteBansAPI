package middleware

import (
	"context"
	"net/http"

	"xmine/litebans-api/api"
)

// AuthStep wraps a StrictHandlerFunc with one authentication/authorization
// check. Steps are small and composable: e.g. RequireJWT resolves identity
// into the context; a later step (e.g. RequireModPermission) can read that
// identity without knowing how it was authenticated.
type AuthStep func(api.StrictHandlerFunc) api.StrictHandlerFunc

// AllowAnonymous is a no-op step for explicitly public operations.
func AllowAnonymous(next api.StrictHandlerFunc) api.StrictHandlerFunc { return next }

// AuthRule overrides the policy's Default chain for a set of operationIds
// that share the same requirement, so a series of similarly-protected
// endpoints doesn't need one copy-pasted rule each.
type AuthRule struct {
	Operations []string
	Chain      []AuthStep
}

// AuthPolicy is a Spring-Security-style authorization table: explicit Rules
// win, any operationId not listed here falls back to Default. If Default is
// left empty, every unlisted operation is rejected outright — the policy
// fails closed even if nobody got around to configuring a default, instead
// of silently allowing everything through.
type AuthPolicy struct {
	Default []AuthStep
	Rules   []AuthRule
}

func denyAll(_ api.StrictHandlerFunc) api.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, nil
	}
}

// Authorize builds the single StrictMiddlewareFunc that enforces policy for
// every operation by composing its chain: Chain[0] runs first.
//
// The operationId -> chain lookup is compiled into a map once here, since
// the policy is fixed at startup and never changes afterwards — every
// request then pays for a map lookup instead of scanning Rules.
func Authorize(policy AuthPolicy) api.StrictMiddlewareFunc {
	byOperation := make(map[string][]AuthStep, len(policy.Rules))
	for _, r := range policy.Rules {
		for _, op := range r.Operations {
			byOperation[op] = r.Chain
		}
	}

	defaultChain := policy.Default
	if len(defaultChain) == 0 {
		defaultChain = []AuthStep{denyAll}
	}

	return func(f api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		chain, ok := byOperation[operationID]
		if !ok {
			chain = defaultChain
		}
		result := f
		for i := len(chain) - 1; i >= 0; i-- {
			result = chain[i](result)
		}
		return result
	}
}
