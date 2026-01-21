package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"gopkg.in/go-jose/go-jose.v2"
)

// RedisClient interface abstracts Redis operations for testing
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
}

// JWKSCache provides Redis-backed caching for JWKS with HTTP fallback.
// Architecture: Redis (L1) -> HTTP Fetch (L2) -> Error
//
// TTL Strategy:
//   - Primary cache: 12 hours (recommended by Staff Engineer)
//   - Rationale: Matches typical JWKS key rotation windows (24-48hrs)
//   - Safety margin: 12hrs provides staleness buffer while capturing rotations
type JWKSCache struct {
	redisClient RedisClient
	httpClient  *http.Client
	jwksURL     string
	cacheTTL    time.Duration
}

// NewJWKSCache creates a new JWKS cache instance.
// Parameters:
//   - redisClient: Redis client for caching (implements RedisClient interface)
//   - jwksURL: URL to fetch JWKS from
//   - cacheTTL: How long to cache JWKS (recommended: 12 hours)
func NewJWKSCache(
	redisClient RedisClient,
	jwksURL string,
	cacheTTL time.Duration,
) *JWKSCache {
	return &JWKSCache{
		redisClient: redisClient,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		jwksURL:  jwksURL,
		cacheTTL: cacheTTL,
	}
}

// GetJWKS retrieves the JSON Web Key Set from cache or HTTP endpoint.
// Implements layered fallback:
//  1. Try Redis cache (fast path)
//  2. On miss, fetch from HTTP endpoint
//  3. Update Redis for future requests
//  4. On any error, return error (caller handles fallback)
func (j *JWKSCache) GetJWKS(ctx context.Context) (*jose.JSONWebKeySet, error) {
	cacheKey := j.GetCacheKey()

	// Attempt Redis cache hit
	val, err := j.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit - deserialize and return
		return j.unmarshalJWKS(val)
	}

	if err != redis.Nil {
		// Redis error (network, timeout, etc.) - log and continue to HTTP
		// In production, would add structured logging here
		// For now, we proceed with HTTP fetch as fallback
	}

	// Cache miss or Redis unavailable - fetch from HTTP endpoint
	keySet, err := j.fetchFromHTTP(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Update Redis cache (best-effort, non-blocking)
	// If cache update fails, we still return the successfully fetched keyset
	if err := j.setCache(ctx, cacheKey, keySet); err != nil {
		// In production, would log cache write failure
		// Not a critical error - next request will fetch fresh if needed
	}

	return keySet, nil
}

// setCache stores the JWKS keyset in Redis with TTL.
func (j *JWKSCache) setCache(
	ctx context.Context,
	key string,
	keySet *jose.JSONWebKeySet,
) error {
	data, err := json.Marshal(keySet)
	if err != nil {
		return fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	return j.redisClient.Set(ctx, key, data, j.cacheTTL).Err()
}

// unmarshalJWKS deserializes cached JWKS data.
func (j *JWKSCache) unmarshalJWKS(data string) (*jose.JSONWebKeySet, error) {
	var keySet jose.JSONWebKeySet
	if err := json.Unmarshal([]byte(data), &keySet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWKS: %w", err)
	}
	return &keySet, nil
}

// getCacheKey returns the Redis key for JWKS caching.
// Uses MD5 hash of URL to support multiple issuers while keeping keys concise.
// Format: "jwks:issuer:{url_hash}"
// Exported for testing purposes
func (j *JWKSCache) GetCacheKey() string {
	hash := md5.Sum([]byte(j.jwksURL))
	return fmt.Sprintf("jwks:issuer:%x", hash)
}

// fetchFromHTTP retrieves JWKS from the configured endpoint.
func (j *JWKSCache) fetchFromHTTP(ctx context.Context) (*jose.JSONWebKeySet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, j.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var keySet jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	return &keySet, nil
}
