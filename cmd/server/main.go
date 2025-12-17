package main

import (
	"bytes"
	"context"
	"encoding/json"
	"es/internal"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

var loadBalancerAddress = flag.String("lb", "http://localhost:8080", "load balancer address")

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("failed to load .env file: %v", err)
	}

	flag.Parse()

	if *loadBalancerAddress == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func main() {
	ports := []int{5001, 5002, 5003}

	for _, port := range ports {
		go runServer(port)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	fmt.Println("Application started!")

	<-signalCh
	fmt.Println("Received shutdown signal, exiting...")
}

func runServer(port int) {
	ctx := context.Background()
	pool := internal.MustDBPool(ctx)

	app := internal.NewApi(pool)
	registerServer(port)

	log.Fatal(app.Listen(fmt.Sprintf(":%d", port)))
}

func registerServer(port int) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(map[string]string{
		"url": fmt.Sprintf("http://localhost:%d", port),
	}); err != nil {
		fmt.Printf("failed to encode payload: %v", err)
		return
	}
	resp, err := http.Post(
		fmt.Sprintf("%s/register", *loadBalancerAddress),
		"application/json",
		bytes.NewReader(buf.Bytes()),
	)

	if err != nil {
		fmt.Printf("failed to connect to load balancer: %v", err)
		return
	}

	if resp.StatusCode != http.StatusCreated {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(resp.Body)
		if err != nil {
			fmt.Printf("failed to read response body from lb: %v", err)
		}
		resp.Body.Close()
		return
	}

	fmt.Println("Server registered with load balancer!")
}
