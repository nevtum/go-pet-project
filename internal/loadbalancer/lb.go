package loadbalancer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
)

type Server struct {
	URL            *url.URL
	proxy          *httputil.ReverseProxy
	IsHealthy      bool
	FailedAttempts int
	mu             sync.Mutex
}

func (s *Server) ReadinessProbe(ctx context.Context) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "/readyz", nil)
	if err != nil {
		fmt.Printf("error creating request: %v\n", err)
		return
	}
	res := httptest.NewRecorder()
	s.ServeHTTP(res, request)

	s.mu.Lock()
	defer s.mu.Unlock()

	if res.Code != http.StatusOK {
		s.IsHealthy = false
		s.FailedAttempts++
		fmt.Printf("server %s is unhealthy\n", s.URL.String())
		return
	}

	fmt.Printf("server %s is healthy\n", s.URL.String())
	s.IsHealthy = true
	s.FailedAttempts = 0
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

type ServerRegistrationRequest struct {
	URL string `json:"url"`
}

type LoadBalancer struct {
	healthCheckInterval time.Duration
	servers             []*Server
	idx                 int
	mu                  sync.RWMutex
	quitCh              chan os.Signal
}

func NewLoadBalancer(healthCheckInterval time.Duration, quitCh chan os.Signal) *LoadBalancer {
	lb := &LoadBalancer{
		healthCheckInterval: healthCheckInterval,
		servers:             []*Server{},
		quitCh:              quitCh,
	}

	return lb
}

func (lb *LoadBalancer) registerServer(rawURL string) error {
	// Validate URL
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	return lb.add(parsedURL)
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check for registration endpoint
	if r.Method == http.MethodPost && r.URL.Path == "/register" {
		lb.handleRegisterEndpoint(w, r)
		return
	}

	// Fallback to load balancing
	server := lb.roundRobinNextServer()

	if server == nil {
		http.Error(w, "No healthy servers available", http.StatusServiceUnavailable)
		return
	}

	fmt.Printf("Proxying request to %s\n", server.URL)
	server.ServeHTTP(w, r)
}

func (lb *LoadBalancer) handleRegisterEndpoint(w http.ResponseWriter, r *http.Request) {
	// Validate content type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	// Parse request body
	var payload ServerRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Register server
	err := lb.registerServer(payload.URL)
	if err != nil {
		switch {
		case err.Error() == fmt.Sprintf("server URL already exists: %s", payload.URL):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "Server registered successfully",
		"url":    payload.URL,
	})
}

func (lb *LoadBalancer) serverHealthCheck(server *Server) {
	ticker := time.NewTicker(lb.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lb.quitCh:
			return
		case <-ticker.C:
			ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
			server.ReadinessProbe(ctx)
			if server.FailedAttempts >= 3 {
				lb.remove(server)
				return
			}
		}
	}
}

func (lb *LoadBalancer) add(parsedURL *url.URL) error {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Check for duplicate servers
	for _, existing := range lb.servers {
		if existing.URL.String() == parsedURL.String() {
			return fmt.Errorf("server URL already exists: %s", parsedURL.String())
		}
	}

	// Create and add new server
	newServer := NewServer(parsedURL)
	lb.servers = append(lb.servers, newServer)

	// Run initial health check
	go lb.serverHealthCheck(newServer)
	return nil

}

func (lb *LoadBalancer) remove(server *Server) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	// Find matching server by url and remove
	for i, s := range lb.servers {
		if s.URL.String() == server.URL.String() {
			lb.servers = append(lb.servers[:i], lb.servers[i+1:]...)
			fmt.Printf("server %s removed from liveness checks\n", server.URL.String())
			return
		}
	}
	panic("server not found for removal")
}

func (lb *LoadBalancer) roundRobinNextServer() *Server {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

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
