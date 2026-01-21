package authentication

import (
	"es/internal/cache"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	IssuerURL    string
	JWKSURL      string
	Redis        cache.RedisConfig
	AuthCacheTTL time.Duration
}

func LoadConfig() Config {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURL := os.Getenv("REDIRECT_URL")
	issuerURL := os.Getenv("ISSUER_URL")
	jwksURL := os.Getenv("JWKS_URL")

	// Redis configuration with defaults
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisDB := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if db, err := strconv.Atoi(dbStr); err == nil {
			redisDB = db
		}
	}

	redisPoolSize := 10
	if poolStr := os.Getenv("REDIS_POOL_SIZE"); poolStr != "" {
		if pool, err := strconv.Atoi(poolStr); err == nil && pool > 0 {
			redisPoolSize = pool
		}
	}

	// Cache TTL: 12 hours recommended by architecture (matches JWKS rotation windows)
	cacheTTL := 12 * time.Hour
	if ttlStr := os.Getenv("CACHE_TTL_HOURS"); ttlStr != "" {
		if hours, err := strconv.Atoi(ttlStr); err == nil && hours > 0 {
			cacheTTL = time.Duration(hours) * time.Hour
		}
	}

	return Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		IssuerURL:    issuerURL,
		JWKSURL:      jwksURL,
		AuthCacheTTL: cacheTTL,
		Redis: cache.RedisConfig{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
			PoolSize: redisPoolSize,
		},
	}
}
