package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"
)

// AuthorityClient checks moderator permissions against the Authority Service's
// GET /rank/{uuid} endpoint, caching responses briefly since permissions change rarely.
type AuthorityClient struct {
	baseURL       string
	internalToken string
	httpClient    *http.Client
	ttl           time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	permissions []string
	expiresAt   time.Time
}

type rankResponse struct {
	Permissions []string `json:"permissions"`
}

func NewAuthorityClient(baseURL, internalToken string, ttl time.Duration) *AuthorityClient {
	return &AuthorityClient{
		baseURL:       baseURL,
		internalToken: internalToken,
		httpClient:    &http.Client{Timeout: 5 * time.Second},
		ttl:           ttl,
		cache:         make(map[string]cacheEntry),
	}
}

// HasPermission reports whether the given player uuid holds the given permission node,
// per Authority Service's /rank/{uuid}. Returns an error if Authority is unreachable.
func (c *AuthorityClient) HasPermission(ctx context.Context, uuid, permission string) (bool, error) {
	permissions, err := c.permissions(ctx, uuid)
	if err != nil {
		return false, err
	}
	return slices.Contains(permissions, permission), nil
}

func (c *AuthorityClient) permissions(ctx context.Context, uuid string) ([]string, error) {
	if perms, ok := c.fromCache(uuid); ok {
		return perms, nil
	}

	url := fmt.Sprintf("%s/rank/%s", c.baseURL, uuid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("authority: build request: %w", err)
	}
	req.Header.Set("X-Internal-Token", c.internalToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("authority: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authority: unexpected status %d", resp.StatusCode)
	}

	var rank rankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rank); err != nil {
		return nil, fmt.Errorf("authority: decode response: %w", err)
	}

	c.store(uuid, rank.Permissions)
	return rank.Permissions, nil
}

func (c *AuthorityClient) fromCache(uuid string) ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.cache[uuid]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.permissions, true
}

func (c *AuthorityClient) store(uuid string, permissions []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[uuid] = cacheEntry{permissions: permissions, expiresAt: time.Now().Add(c.ttl)}
}
