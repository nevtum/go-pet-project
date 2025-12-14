package internal

import (
	"context"
	"es/internal/util"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func MustDBConn(ctx context.Context) *pgx.Conn {
	DBConnString := os.Getenv("PG_CONNSTRING")
	return util.Must(pgx.Connect(ctx, DBConnString))
}

func MustDBPool(ctx context.Context) *pgxpool.Pool {
	DBConnString := os.Getenv("PG_CONNSTRING")
	return util.Must(DBPool(ctx, DBConnString))
}

func DBPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	// Initialize the connection pool
	pool, err := pgxpool.New(ctx, connString)

	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %s", err)
	}

	// Verify the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Unable to ping database: %s", err)
	}

	fmt.Println("Connected to PostgreSQL database!")

	return pool, nil
}
