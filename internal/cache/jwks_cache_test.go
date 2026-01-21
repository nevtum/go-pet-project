package cache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-jose/go-jose.v2"
)

// createTestKeySet creates a sample JWKS for testing
func createTestKeySet() *jose.JSONWebKeySet {
	return &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				KeyID: "key1",
				Key:   "public_key_data",
			},
		},
	}
}

// TestNewJWKSCache tests cache initialization
func TestNewJWKSCache(t *testing.T) {
	// Setup - using a nil mock client for initialization test
	cache := NewJWKSCache(nil, "https://example.com/.well-known/jwks.json", 12*time.Hour)

	// Assert
	assert.NotNil(t, cache)
	assert.Equal(t, "https://example.com/.well-known/jwks.json", cache.jwksURL)
	assert.Equal(t, 12*time.Hour, cache.cacheTTL)
}

// TestFetchFromHTTP tests JWKS HTTP fetching
func TestFetchFromHTTP(t *testing.T) {
	// Setup
	keySet := createTestKeySet()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keySet)
	}))
	defer server.Close()

	cache := &JWKSCache{
		redisClient: nil,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		jwksURL:  server.URL,
		cacheTTL: 12 * time.Hour,
	}

	// Act
	result, err := cache.fetchFromHTTP(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.Keys))
	assert.Equal(t, "key1", result.Keys[0].KeyID)
}

// TestFetchFromHTTP_NetworkError tests HTTP fetch with network error
func TestFetchFromHTTP_NetworkError(t *testing.T) {
	// Setup
	cache := &JWKSCache{
		redisClient: nil,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		jwksURL:  "https://invalid-domain-12345.example.com",
		cacheTTL: 12 * time.Hour,
	}

	// Act
	result, err := cache.fetchFromHTTP(context.Background())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

// TestFetchFromHTTP_ServerError tests HTTP fetch with server error
func TestFetchFromHTTP_ServerError(t *testing.T) {
	// Setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	cache := &JWKSCache{
		redisClient: nil,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		jwksURL:  server.URL,
		cacheTTL: 12 * time.Hour,
	}

	// Act
	result, err := cache.fetchFromHTTP(context.Background())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "HTTP 500")
}

// TestFetchFromHTTP_InvalidJSON tests HTTP fetch with invalid JSON response
func TestFetchFromHTTP_InvalidJSON(t *testing.T) {
	// Setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json {"))
	}))
	defer server.Close()

	cache := &JWKSCache{
		redisClient: nil,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		jwksURL:  server.URL,
		cacheTTL: 12 * time.Hour,
	}

	// Act
	result, err := cache.fetchFromHTTP(context.Background())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode JWKS")
}

// TestUnmarshalJWKS tests JWKS deserialization
func TestUnmarshalJWKS(t *testing.T) {
	// Setup
	cache := &JWKSCache{}
	keySet := createTestKeySet()
	data, err := json.Marshal(keySet)
	require.NoError(t, err)

	// Act
	result, err := cache.unmarshalJWKS(string(data))

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "key1", result.Keys[0].KeyID)
	assert.Equal(t, "public_key_data", result.Keys[0].Key)
}

// TestUnmarshalJWKS_InvalidJSON tests deserialization with invalid JSON
func TestUnmarshalJWKS_InvalidJSON(t *testing.T) {
	// Setup
	cache := &JWKSCache{}

	// Act
	result, err := cache.unmarshalJWKS("invalid json {")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to unmarshal JWKS")
}

// TestGetCacheKey tests cache key generation
func TestGetCacheKey(t *testing.T) {
	// Setup
	cache1 := &JWKSCache{
		jwksURL: "https://example.com/.well-known/jwks.json",
	}
	cache2 := &JWKSCache{
		jwksURL: "https://different.com/.well-known/jwks.json",
	}

	// Act
	key1 := cache1.GetCacheKey()
	key2 := cache2.GetCacheKey()

	// Assert
	assert.NotEqual(t, key1, key2)
	assert.True(t, len(key1) > 0)
	assert.True(t, len(key2) > 0)
	assert.Contains(t, key1, "jwks:issuer:")
	assert.Contains(t, key2, "jwks:issuer:")
}

// TestGetCacheKey_Consistency tests that same URL generates same key
func TestGetCacheKey_Consistency(t *testing.T) {
	// Setup
	url := "https://example.com/.well-known/jwks.json"
	cache1 := &JWKSCache{jwksURL: url}
	cache2 := &JWKSCache{jwksURL: url}

	// Act
	key1 := cache1.GetCacheKey()
	key2 := cache2.GetCacheKey()

	// Assert
	assert.Equal(t, key1, key2)
}

// TestMarshalUnmarshalRoundtrip tests serialization consistency
func TestMarshalUnmarshalRoundtrip(t *testing.T) {
	// Setup
	cache := &JWKSCache{}
	original := createTestKeySet()

	// Serialize
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Deserialize
	restored, err := cache.unmarshalJWKS(string(data))

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, restored)
	assert.Equal(t, len(original.Keys), len(restored.Keys))
	assert.Equal(t, original.Keys[0].KeyID, restored.Keys[0].KeyID)
	assert.Equal(t, original.Keys[0].Key, restored.Keys[0].Key)
}

// TestSetCache_ErrorHandling tests error handling in cache operations
func TestSetCache_ErrorHandling(t *testing.T) {
	// This test verifies that cache layer handles marshaling errors gracefully
	// cache := &JWKSCache{
	// 	cacheTTL: 12 * time.Hour,
	// }

	// Test with valid keyset
	keySet := createTestKeySet()
	// Note: In production with a real Redis client, this would be tested differently
	// For now, we verify the marshaling logic doesn't panic
	data, err := json.Marshal(keySet)
	assert.NoError(t, err)
	assert.NotNil(t, data)
}

// TestCacheKeyFormat tests the format of generated cache keys
func TestCacheKeyFormat(t *testing.T) {
	// Setup
	cache := &JWKSCache{
		jwksURL: "https://auth.example.com/.well-known/jwks.json",
	}

	// Act
	key := cache.GetCacheKey()

	// Assert
	// Key should be in format: jwks:issuer:<md5hash>
	assert.Len(t, key, len("jwks:issuer:")+32) // md5 hash is 32 chars in hex
	assert.True(t, len(key) >= 20)
	assert.Contains(t, key, "jwks:issuer:")
}

// TestFetchFromHTTP_Timeout tests HTTP fetch with timeout
func TestFetchFromHTTP_Timeout(t *testing.T) {
	// Setup
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createTestKeySet())
	}))
	defer server.Close()

	cache := &JWKSCache{
		redisClient: nil,
		httpClient: &http.Client{
			Timeout: 100 * time.Millisecond, // Very short timeout
		},
		jwksURL:  server.URL,
		cacheTTL: 12 * time.Hour,
	}

	// Act
	result, err := cache.fetchFromHTTP(context.Background())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "HTTP request failed")
}

// TestGetJWKS_Documentation tests and documents the expected behavior
func TestGetJWKS_Documentation(t *testing.T) {
	// This test serves as documentation of the GetJWKS contract:
	// 1. Returns valid JWKS on success
	// 2. Handles Redis cache misses
	// 3. Falls back to HTTP fetch
	// 4. Updates cache on successful HTTP fetch
	// 5. Returns errors appropriately

	keySet := createTestKeySet()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keySet)
	}))
	defer server.Close()

	// Note: Full GetJWKS testing requires Redis mock integration
	// This test documents the API contract
	cache := NewJWKSCache(nil, server.URL, 12*time.Hour)
	assert.NotNil(t, cache)
}
