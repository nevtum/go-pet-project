package main

import (
	"context"
	"es/internal"
	"es/internal/es"
	v1 "es/internal/inventory/v1"
	v2 "es/internal/inventory/v2"
	"es/internal/util"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("failed to load .env file: %v", err)
	}
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*120)

	pool := internal.MustDBPool(ctx)
	stream := es.NewEventStream(pool)
	var batchSize int64 = 25

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		sub := es.NewSubscription(
			v1.NewProjection(pool),
			batchSize,
			time.Second*5,
		)
		util.MustSucceed(sub.Listen(ctx, stream))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		sub := es.NewSubscription(
			v2.NewProjection(pool),
			batchSize,
			time.Second*2,
		)

		util.MustSucceed(sub.Listen(ctx, stream))
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Context timed out")
	case <-stopChan:
		fmt.Println("Received signal to stop")
		cancel()
	}

	wg.Wait()
}
