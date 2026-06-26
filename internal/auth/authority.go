package auth

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"
	"xmine/litebans-api/clients/authority"
)

// AuthorityClient checks moderator permissions against the Authority Service's
// GET /rank/{uuid} endpoint, caching responses briefly since permissions change rarely.
type AuthorityClient struct {
	client *authority.ClientWithResponses
	ttl    time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	permissions []string
	expiresAt   time.Time
}

func NewAuthorityClient(baseURL, internalToken string, ttl time.Duration) *AuthorityClient {
	client, err := authority.NewInternalClient(baseURL, internalToken)
	if err != nil {
		panic(fmt.Errorf("authority: build client: %w", err))
	}
	return &AuthorityClient{
		client: client,
		ttl:    ttl,
		cache:  make(map[string]cacheEntry),
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

	resp, err := c.client.GetRankByUUIDWithResponse(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("authority: request failed: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("authority: unexpected status %s", resp.Status())
	}

	c.store(uuid, resp.JSON200.Permissions)
	return resp.JSON200.Permissions, nil
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
