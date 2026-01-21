package cache

import "github.com/prometheus/client_golang/prometheus"

// TODO: figure out why custom metrics aren't registered
// when /metrics endpoint is called
var (
	// Metrics declaration
	cacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_cache_hits_total",
			Help: "Total number of JWKS cache hits",
		},
		[]string{"source"},
	)

	cacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jwks_cache_misses_total",
			Help: "Total number of JWKS cache misses",
		},
		[]string{"source"},
	)
)

type cacheMetrics struct{}

func (m *cacheMetrics) IncrementCacheHit() {
	cacheHits.With(prometheus.Labels{
		"source": "redis",
	}).Inc()
}

func (m *cacheMetrics) IncrementCacheMiss() {
	cacheMisses.With(prometheus.Labels{
		"source": "redis",
	}).Inc()
}

var metrics = &cacheMetrics{}
