package main

import (
	"context"
	"es/internal/es"
	v1 "es/internal/inventory/v1"
	v2 "es/internal/inventory/v2"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func dbPool(ctx context.Context) *pgxpool.Pool {
	// Initialize the connection pool
	pool, err := pgxpool.New(ctx, "postgres://myuser:mypassword@localhost:15432/mydatabase")

	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}

	// Verify the connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Unable to ping database:", err)
	}

	fmt.Println("Connected to PostgreSQL database!")

	return pool
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*120)

	pool := dbPool(ctx)
	stream := es.NewEventStream(pool)
	var batchSize int64 = 25

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGTERM, syscall.SIGINT)

	wg := sync.WaitGroup{}

	go func() {
		wg.Add(1)
		defer wg.Done()
		if err := es.NewSubscription(
			v1.NewProjection(pool),
			batchSize,
			time.Second*5,
		).Listen(ctx, stream); err != nil {
			log.Fatal("Unable to listen to event stream:", err)
		}
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()
		if err := es.NewSubscription(
			v2.NewProjection(pool),
			batchSize,
			time.Second*2,
		).Listen(ctx, stream); err != nil {
			log.Fatal("Unable to listen to event stream:", err)
		}
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
