package main

import (
	"context"
	"es/internal/api"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

func runServer(port int) {
	pool := dbPool()
	repo := api.NewPGCartRepository(pool)
	h := api.NewShoppingCartHandler(repo)

	http.ListenAndServe(fmt.Sprintf(":%d", port), h)
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
