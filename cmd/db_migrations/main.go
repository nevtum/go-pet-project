package main

import (
	"context"
	"es/internal"
	"es/internal/util"
	"os"

	"fmt"
)

func main() {
	fmt.Println("Migrating database...")
	ctx := context.Background()
	conn := internal.MustDBConn(ctx)
	defer conn.Close(ctx)

	sqlFile := util.Must(os.ReadFile("cmd/db_migrations/create_events_table.sql"))
	_ = util.Must(conn.Exec(ctx, string(sqlFile)))

	fmt.Println("Migration executed successfully.")
}
