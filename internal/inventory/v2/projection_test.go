package v2_test

import (
	"context"
	"es/internal/api"
	"es/internal/es"
	v2 "es/internal/inventory/v2"
	"es/internal/util"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

type testContext struct {
	db         string
	pool       *pgxpool.Pool
	projection *v2.Projection
	ctx        context.Context
	cancel     context.CancelFunc
}

func setupTestContext(t *testing.T) *testContext {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	defaultConnStr := "postgres://myuser:mypassword@localhost:15432/postgres"
	defaultConn, err := pgx.Connect(ctx, defaultConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to default database: %v", err)
	}
	defer defaultConn.Close(ctx)

	// Create a unique database
	dbName := fmt.Sprintf("testdb_%d", time.Now().UnixNano())

	_, err = defaultConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	connStr := fmt.Sprintf("postgres://myuser:mypassword@localhost:15432/%s", dbName)
	pool := util.Must(pgxpool.New(ctx, connStr))

	projection := v2.NewProjection(pool)
	assert.NoError(t, projection.ApplyMigration(ctx))

	return &testContext{
		db:         dbName,
		pool:       pool,
		projection: projection,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func TestProjectionV2(t *testing.T) {
	t.Run("when item added to cart", func(t *testing.T) {
		tc := setupTestContext(t)

		itemID := 42
		cartID := 1001

		assert.NoError(t, tc.projection.Apply(tc.ctx,
			es.Event{
				Type:        api.ItemAddedToCart,
				AggregateID: cartID,
				Data:        map[string]int{"item_id": itemID},
				Position:    1,
			},
			es.Event{
				Type:        api.ItemAddedToCart,
				AggregateID: cartID,
				Data:        map[string]int{"item_id": itemID},
				Position:    2,
			},
		))

		assert.NoError(t, tc.projection.Apply(tc.ctx,
			es.Event{
				Type:        api.ItemRemovedFromCart,
				AggregateID: cartID,
				Data:        map[string]int{"item_id": itemID},
				Position:    3,
			},
			es.Event{
				Type:        api.ItemAddedToCart,
				AggregateID: cartID,
				Data:        map[string]int{"item_id": itemID},
				Position:    4,
			},
		))

		// Verify cart item quantity
		var quantity int
		err := tc.pool.QueryRow(tc.ctx, `
			SELECT quantity
			FROM inventory_v2.cart_items
			WHERE cart_id = $1 AND item_id = $2
		`, cartID, itemID).Scan(&quantity)
		assert.NoError(t, err)
		assert.Equal(t, 2, quantity, "Quantity should be 2 after adding same item twice")
	})

	t.Run("when item removed from cart", func(t *testing.T) {
		tc := setupTestContext(t)

		itemID := 42
		cartID := 1001
		cartID2 := 1002

		assert.NoError(t, tc.projection.Apply(tc.ctx, es.Event{
			Type:        api.ItemAddedToCart,
			AggregateID: cartID,
			Data:        map[string]int{"item_id": itemID},
			Position:    1,
		}))

		assert.NoError(t, tc.projection.Apply(tc.ctx,
			es.Event{
				Type:        api.ItemAddedToCart,
				AggregateID: cartID2,
				Data:        map[string]int{"item_id": itemID},
				Position:    2,
			},
			es.Event{
				Type:        api.ItemRemovedFromCart,
				AggregateID: cartID,
				Data:        map[string]int{"item_id": itemID},
				Position:    3,
			}))

		// Verify cart item quantity
		var quantity int
		err := tc.pool.QueryRow(tc.ctx, `
			SELECT quantity
			FROM inventory_v2.cart_items
			WHERE cart_id = $1 AND item_id = $2
		`, cartID, itemID).Scan(&quantity)
		assert.NoError(t, err)
		assert.Equal(t, 0, quantity, "Quantity should be 0 after removing item twice")
	})
}
