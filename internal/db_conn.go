package internal

import (
	"context"
	"es/internal/util"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// DBConnString is the connection string for the database.
	// Warning: Do not hardcode credentials in production code.
	DBConnString = "postgres://myuser:mypassword@localhost:15432/mydatabase"
)

func MustDBConn(ctx context.Context) *pgx.Conn {
	return util.Must(pgx.Connect(ctx, DBConnString))
}

func MustDBPool(ctx context.Context) *pgxpool.Pool {
	return util.Must(DBPool(ctx))
}

func DBPool(ctx context.Context) (*pgxpool.Pool, error) {
	// Initialize the connection pool
	pool, err := pgxpool.New(ctx, DBConnString)

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
