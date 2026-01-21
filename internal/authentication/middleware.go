package authentication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"es/internal/cache"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gopkg.in/go-jose/go-jose.v2"
)

// AuthMiddleware creates an authentication middleware with Redis-backed JWKS caching.
// The middleware:
//  1. Extracts JWT from Authorization header
//  2. Validates claims (expiration, issuer, audience)
//  3. Retrieves JWKS from Redis cache or HTTP endpoint
//  4. Verifies token signature against cached keys
func AuthMiddleware(cfg *Config) (fiber.Handler, error) {
	// Initialize Redis client for JWKS caching
	redisClient, err := cache.NewRedisClient(cache.ClientConfig{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: 5,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis client: %w", err)
	}

	jwksCache := cache.NewJWKSCache(redisClient, cfg.JWKSURL, cfg.Redis.AuthCacheTTL)
	verifyer := newVerifyer(cfg, jwksCache)

	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")
		if tokenString == "" {
			return c.Status(http.StatusUnauthorized).SendString("missing authorization token")
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		claims, err := verifyer.verifyToken(c.Context(), tokenString)
		if err != nil {
			return c.Status(http.StatusUnauthorized).SendString(fmt.Sprintf("Invalid token: %v", err))
		}

		// Add claims to request context
		c.Locals("user", claims)

		return c.Next()
	}, nil
}

type verifyer struct {
	cfg       *Config
	jwksCache *cache.JWKSCache
}

func newVerifyer(cfg *Config, jwksCache *cache.JWKSCache) *verifyer {
	return &verifyer{
		cfg:       cfg,
		jwksCache: jwksCache,
	}
}

func (v *verifyer) verifyToken(ctx context.Context, tokenString string) (jwt.MapClaims, error) {
	// Parse the token without verifying signature first to get key ID
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	// Extract claims and header
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Validate standard claims
	if err := v.validateClaims(claims); err != nil {
		return nil, err
	}

	// Find the correct key for verification
	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("no key ID found in token")
	}

	// Fetch JWKS from cache (Redis) with HTTP fallback
	keySet, err := v.jwksCache.GetJWKS(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %v", err)
	}

	// Verify token signature
	return verifyTokenSignature(tokenString, keyID, keySet)
}

func (v *verifyer) validateClaims(claims jwt.MapClaims) error {
	// Validate token expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return errors.New("token has expired")
		}
	} else {
		return errors.New("no expiration claim found")
	}

	// Validate issuer
	if iss, ok := claims["iss"].(string); !ok || iss != v.cfg.IssuerURL {
		return errors.New("invalid token issuer")
	}

	// Validate audience (client ID)
	if aud, ok := claims["client_id"].(string); !ok || aud != v.cfg.ClientID {
		return errors.New("invalid token audience")
	}

	return nil
}

func fetchJWKS(jwksURL string) (*jose.JSONWebKeySet, error) {
	// In production, implement proper HTTP client with timeout and error handling
	resp, err := http.Get(jwksURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var keySet jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keySet); err != nil {
		return nil, err
	}

	return &keySet, nil
}

func verifyTokenSignature(tokenString, keyID string, keySet *jose.JSONWebKeySet) (jwt.MapClaims, error) {
	// Find the key with the matching key ID
	var signingKey *jose.JSONWebKey
	for _, key := range keySet.Keys {
		if key.KeyID == keyID {
			signingKey = &key
			break
		}
	}

	if signingKey == nil {
		return nil, errors.New("no matching key found")
	}

	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Return the public key
		return signingKey.Key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token verification failed: %v", err)
	}

	// Extract and return claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
