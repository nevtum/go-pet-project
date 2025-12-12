package loadbalancer

import (
	"context"
	"es/internal/util"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Server struct {
	URL       *url.URL
	proxy     *httputil.ReverseProxy
	IsHealthy bool
	mu        sync.Mutex
}

func (s *Server) ReadinessProbe(ctx context.Context) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "/readyz", nil)
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		return
	}
	res := httptest.NewRecorder()
	s.ServeHTTP(res, request)

	if res.Code != http.StatusOK {
		s.IsHealthy = false
		fmt.Printf("server %s is unhealthy\n", s.URL.String())
		return
	}

	fmt.Printf("server %s is healthy\n", s.URL.String())
	s.IsHealthy = true
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func NewServer(URL *url.URL) *Server {
	return &Server{
		URL:       URL,
		proxy:     httputil.NewSingleHostReverseProxy(URL),
		IsHealthy: true,
	}
}

type LoadBalancer struct {
	healthCheckInterval time.Duration
	servers             []*Server
	idx                 int
	mu                  sync.Mutex
}

func MustNewLoadBalancer(healthCheckInterval time.Duration, urls ...string) *LoadBalancer {
	return util.Must(NewLoadBalancer(healthCheckInterval, urls...))
}

func NewLoadBalancer(healthCheckInterval time.Duration, urls ...string) (*LoadBalancer, error) {
	servers := []*Server{}
	for _, rawURL := range urls {
		URL, err := url.Parse(rawURL)
		if err != nil {
			return nil, err
		}

		servers = append(servers, NewServer(URL))
	}

	lb := &LoadBalancer{
		healthCheckInterval: healthCheckInterval,
		servers:             servers,
	}

	return lb, nil
}

func (lb *LoadBalancer) RunHealthCheckLoop(ctx context.Context) {
	for _, server := range lb.servers {
		go lb.serverHealthCheck(ctx, server)
	}
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server := lb.roundRobinNextServer()

	if server == nil {
		http.Error(w, "No healthy servers available", http.StatusServiceUnavailable)
		return
	}

	fmt.Printf("Proxying request to %s\n", server.URL)
	server.ServeHTTP(w, r)
}

func (lb *LoadBalancer) serverHealthCheck(ctx context.Context, server *Server) {
	ticker := time.NewTicker(lb.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lb.mu.Lock()
			server.ReadinessProbe(ctx)
			lb.mu.Unlock()
		}
	}
}

func (lb *LoadBalancer) roundRobinNextServer() *Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for i := 0; i < len(lb.servers); i++ {
		idx := lb.idx % len(lb.servers)
		next := lb.servers[idx]
		lb.idx++

		if next.IsHealthy {
			return next
		}
	}

	return nil
}
