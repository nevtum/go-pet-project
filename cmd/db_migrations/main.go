package main

import (
	"context"
	"os"

	"log"

	"fmt"

	"github.com/jackc/pgx/v5"
)

func main() {
	fmt.Println("Migrating database...")
	// Database connection string
	connString := "postgres://myuser:mypassword@localhost:15432/mydatabase"

	// Connect to the database
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	// Read SQL from file
	sqlFile, err := os.ReadFile("cmd/db_migrations/create_events_table.sql")
	if err != nil {
		log.Fatalf("Unable to read SQL file: %v\n", err)
	}

	// Execute the SQL
	_, err = conn.Exec(context.Background(), string(sqlFile))
	if err != nil {
		log.Fatalf("Unable to execute migration: %v\n", err)
	}

	fmt.Println("Migration executed successfully.")
}
