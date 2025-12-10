package main

import (
	"context"
	"es/internal/loadbalancer"
	"es/internal/util"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func runLoadBalancer(ctx context.Context, urls ...string) {
	lb := loadbalancer.MustNewLoadBalancer(2*time.Second, urls...)
	lb.RunHealthCheckLoop(ctx)

	util.MustSucceed(http.ListenAndServe(":8080", lb))
}

func main() {
	urls := []string{
		"http://localhost:5001",
		"http://localhost:5002",
		"http://localhost:5003",
	}

	ctx, cancel := context.WithCancel(context.Background())
	go runLoadBalancer(ctx, urls...)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	fmt.Println("Load Balancer started on port 8080")

	<-signalCh
	cancel()
}
