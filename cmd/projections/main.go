package main

import (
	"context"
	"es/internal/es"
	"es/internal/inventory"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func dbPool() *pgxpool.Pool {
	// Initialize the connection pool
	ctx := context.Background()
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
	pool := dbPool()
	stream := es.NewEventStream(pool, 10000)
	projectionWriter := inventory.NewProjection(pool)
	ctx := context.Background()
	err := stream.Subscribe(ctx, projectionWriter)

	if err != nil {
		log.Fatal("Unable to subscribe to event stream:", err)
	}

	fmt.Println("Hello, World!")
}
