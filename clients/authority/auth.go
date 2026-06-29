package authority

import (
	"context"
	"net/http"
	"xmine/litebans-api/internal/requestid"
)

func WithInternalToken(token string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-Internal-Token", token)
		return nil
	}
}

// NewInternalClient builds a client for internal calls (e.g. GET /rank/{uuid}), which use
// X-Internal-Token rather than a player's Bearer JWT. The HTTP client propagates the
// inbound request's X-Request-Id (set by middleware.Observability) onto every outgoing
// request, so the call chain shares one request id across services' logs.
func NewInternalClient(baseURL, internalToken string) (*ClientWithResponses, error) {
	httpClient := &http.Client{Transport: requestid.PropagatingTransport{}}
	return NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(WithInternalToken(internalToken)))
}
