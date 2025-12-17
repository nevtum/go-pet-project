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

func runLoadBalancer(quitCh chan os.Signal) {
	lb := loadbalancer.NewLoadBalancer(2*time.Second, quitCh)
	util.MustSucceed(http.ListenAndServe(":8080", lb))
}

func main() {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	go runLoadBalancer(signalCh)
	fmt.Println("Load Balancer started on port 8080")

	<-signalCh
	fmt.Println("Load Balancer stopped")
}
