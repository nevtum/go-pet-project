package main

import (
	"es/internal/loadbalancer"
	"es/internal/util"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func runLoadBalancer(quitCh chan os.Signal, urls ...string) {
	lb := loadbalancer.NewLoadBalancer(2*time.Second, quitCh)
	for _, rawURL := range urls {
		util.MustSucceed(lb.RegisterServer(rawURL))
	}

	util.MustSucceed(http.ListenAndServe(":8080", lb))
}

func main() {
	urls := []string{
		"http://localhost:5001",
		"http://localhost:5002",
		"http://localhost:5003",
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	go runLoadBalancer(signalCh, urls...)
	fmt.Println("Load Balancer started on port 8080")

	<-signalCh
	fmt.Println("Load Balancer stopped")
}
