package main

import (
	"context"
	"es/internal/es"
	"es/internal/inventory"
	"fmt"
	"log"

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

	pool := dbPool(ctx)
	stream := es.NewEventStream(pool)
	projectionWriter := inventory.NewProjection(pool)
	subscription := es.NewSubscription(projectionWriter, 25)

	if err := subscription.Listen(ctx, stream); err != nil {
		log.Fatal("Unable to listen to event stream:", err)
	}
}
