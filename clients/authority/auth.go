package authority

import (
	"context"
	"net/http"
)

func WithInternalToken(token string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-Internal-Token", token)
		return nil
	}
}

// NewInternalClient builds a client for internal calls (e.g. GET /rank/{uuid}), which use
// X-Internal-Token rather than a player's Bearer JWT.
func NewInternalClient(baseURL, internalToken string) (*ClientWithResponses, error) {
	return NewClientWithResponses(baseURL, WithRequestEditorFn(WithInternalToken(internalToken)))
}
