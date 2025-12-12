package main

import (
	"context"
	"es/internal"
	"es/internal/checkout"
	"es/internal/es"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func runServer(port int) {
	ctx := context.Background()
	pool := internal.MustDBPool(ctx)
	repo := checkout.NewPGCartRepository(pool)
	stream := es.NewEventStream(pool)
	app := checkout.NewShoppingCartHandler(repo, stream)

	log.Fatal(app.Listen(fmt.Sprintf(":%d", port)))
}

func main() {
	go runServer(5001)
	go runServer(5002)
	go runServer(5003)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	fmt.Println("Application started!")

	<-signalCh
	fmt.Println("Received shutdown signal, exiting...")
}
