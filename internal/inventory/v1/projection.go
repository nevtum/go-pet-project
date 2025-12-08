package v1

import (
	"context"
	"errors"
	"es/internal/api"
	"es/internal/es"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Projection struct {
	pool *pgxpool.Pool
}

func NewProjection(pool *pgxpool.Pool) *Projection {
	return &Projection{pool: pool}
}

func (p *Projection) Name() string {
	return "inventory_v1"
}

func (p *Projection) SubscribedEvents() []es.EventType {
	return []es.EventType{
		api.ItemAddedToCart,
		api.ItemRemovedFromCart,
		api.CartCheckedOut,
	}
}

func (p *Projection) ApplyMigration(ctx context.Context) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS inventory_v1;`)

	// Create cart_items table
	tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS inventory_v1.cart_items (
			cart_id INTEGER NOT NULL,
			item_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL,
			checked_out BOOLEAN NOT NULL,
			PRIMARY KEY (cart_id, item_id)
		);

		-- Index to help with querying total quantity of sold items by item_id
		CREATE INDEX IF NOT EXISTS idx_inventory_item_sold
		ON inventory_v1.cart_items (item_id, checked_out)
		WHERE checked_out = true;
	`)

	// Create last_processed_position table
	tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS inventory_v1.last_processed_position (
			position INTEGER NOT NULL,
			CONSTRAINT single_row CHECK (position >= 0)
		);

		-- Ensure only one row exists
		INSERT INTO inventory_v1.last_processed_position (position)
		SELECT 0
		WHERE NOT EXISTS (
			SELECT 1 FROM inventory_v1.last_processed_position
		);
	`)

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (p *Projection) LatestPosition(ctx context.Context) (int64, error) {
	var position int64
	err := p.pool.QueryRow(ctx, `
		SELECT position
		FROM inventory_v1.last_processed_position
		LIMIT 1
	`).Scan(&position)
	if err != nil {
		return 0, fmt.Errorf("read latest position: %w", err)
	}
	return position, nil
}

// TODO: Resolve bugs where in some cases checked out carts are
// not properly handled and lead to incorrect inventory levels.
func (p *Projection) Apply(ctx context.Context, events ...es.Event) error {
	if len(events) == 0 {
		return nil
	}

	// Step 1: Analyze all events
	// Map of (cart_id, item_id) -> net quantity change
	itemChanges := make(map[[2]int]int)
	// Set of cart_ids that have been checked out
	checkedOutCarts := make(map[int]bool)
	// Track max position for tracking processed events
	maxPosition := int64(0)

	for _, event := range events {
		if event.Position > maxPosition {
			maxPosition = event.Position
		}

		switch event.Type {
		case api.ItemAddedToCart:
			itemID, err := extractItemID(event.Data)
			if err != nil {
				return fmt.Errorf("extract item_id from ItemAddedToCart event: %w", err)
			}
			key := [2]int{event.AggregateID, itemID}
			itemChanges[key]++
		case api.ItemRemovedFromCart:
			itemID, err := extractItemID(event.Data)
			if err != nil {
				return fmt.Errorf("extract item_id from ItemRemovedFromCart event: %w", err)
			}
			key := [2]int{event.AggregateID, itemID}
			itemChanges[key]--
		case api.CartCheckedOut:
			checkedOutCarts[event.AggregateID] = true
		}
	}

	// Step 2: Open a db transaction
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 3 & 4: For each item change, update or insert the record in the database
	for key, quantityChange := range itemChanges {
		cartID, itemID := key[0], key[1]
		isCheckedOut := checkedOutCarts[cartID]

		// Try to update existing record
		result, err := tx.Exec(ctx, `
			UPDATE inventory_v1.cart_items
			SET quantity = quantity + $1, checked_out = checked_out OR $2
			WHERE cart_id = $3 AND item_id = $4
		`, quantityChange, isCheckedOut, cartID, itemID)
		if err != nil {
			return fmt.Errorf("update cart_items: %w", err)
		}

		// If no rows were updated, insert a new record
		if result.RowsAffected() == 0 {
			_, err := tx.Exec(ctx, `
				INSERT INTO inventory_v1.cart_items (cart_id, item_id, quantity, checked_out)
				VALUES ($1, $2, $3, $4)
			`, cartID, itemID, quantityChange, isCheckedOut)
			if err != nil {
				return fmt.Errorf("insert cart_items: %w", err)
			}
		}
	}

	// Update the last processed position
	_, err = tx.Exec(ctx, `
		UPDATE inventory_v1.last_processed_position
		SET position = $1
	`, maxPosition)
	if err != nil {
		return fmt.Errorf("update last_processed_position: %w", err)
	}

	// Step 5: Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// extractItemID extracts the item_id from event data.
// It handles both map[string]int (direct type) and map[string]any (JSON deserialized).
func extractItemID(data any) (int, error) {
	// Try map[string]int first (direct type)
	if dataMap, ok := data.(map[string]int); ok {
		if id, ok := dataMap["item_id"]; ok {
			return id, nil
		}
		return 0, errors.New("missing item_id in event data")
	}

	// Try map[string]any (for JSON deserialized events)
	if dataAny, ok := data.(map[string]any); ok {
		if v, ok := dataAny["item_id"]; ok {
			switch id := v.(type) {
			case float64:
				return int(id), nil
			case int:
				return id, nil
			}
		}
		return 0, errors.New("missing or invalid item_id in event data")
	}

	return 0, errors.New("invalid event data type")
}
