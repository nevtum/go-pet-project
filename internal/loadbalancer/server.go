package loadbalancer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
)

type Server struct {
	*httputil.ReverseProxy
	URL            *url.URL
	IsHealthy      bool
	FailedAttempts int
	mu             sync.Mutex
}

func NewServer(URL *url.URL) *Server {
	return &Server{
		ReverseProxy: httputil.NewSingleHostReverseProxy(URL),
		URL:          URL,
		IsHealthy:    true,
	}
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
