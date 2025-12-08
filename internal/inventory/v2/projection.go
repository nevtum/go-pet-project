package v2

import (
	"context"
	"errors"
	"es/internal/api"
	"es/internal/es"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Projection struct {
	pool *pgxpool.Pool
}

func NewProjection(pool *pgxpool.Pool) *Projection {
	return &Projection{pool: pool}
}

func (p *Projection) Name() string {
	return "inventory_v2"
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

	defer func(ctx context.Context) {
		_ = tx.Rollback(ctx)
	}(ctx)

	if _, err := tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS inventory_v2;`); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Create carts table
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS inventory_v2.carts (
			cart_id INTEGER PRIMARY KEY,
			checked_out BOOLEAN NOT NULL DEFAULT FALSE
		);
	`); err != nil {
		return fmt.Errorf("create carts table: %w", err)
	}

	// Create cart_items table with foreign key to carts
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS inventory_v2.cart_items (
			cart_id INTEGER NOT NULL,
			item_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL,
			PRIMARY KEY (cart_id, item_id),
			CONSTRAINT fk_cart_items_cart_id
			FOREIGN KEY (cart_id) REFERENCES inventory_v2.carts(cart_id)
		);
	`); err != nil {
		return fmt.Errorf("create cart_items table: %w", err)
	}

	// Create last_processed_position table
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS inventory_v2.last_processed_position (
			position INTEGER NOT NULL,
			CONSTRAINT single_row CHECK (position >= 0)
		);

		-- Ensure only one row exists
		INSERT INTO inventory_v2.last_processed_position (position)
		SELECT 0
		WHERE NOT EXISTS (
			SELECT 1 FROM inventory_v2.last_processed_position
		);
	`); err != nil {
		return fmt.Errorf("create last_processed_position table: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (p *Projection) LatestPosition(ctx context.Context) (int64, error) {
	var position int64
	err := p.pool.QueryRow(ctx, `
		SELECT position
		FROM inventory_v2.last_processed_position
		LIMIT 1
	`).Scan(&position)
	if err != nil {
		return 0, fmt.Errorf("read latest position: %w", err)
	}
	return position, nil
}

func (p *Projection) Apply(ctx context.Context, events ...es.Event) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func(ctx context.Context) {
		_ = tx.Rollback(ctx)
	}(ctx)

	maxPosition := int64(0)

	// Process events one by one
	for _, event := range events {
		if event.Position > maxPosition {
			maxPosition = event.Position
		}

		switch event.Type {
		case api.ItemAddedToCart:
			if err := p.handleItemAddedToCart(ctx, tx, event); err != nil {
				return fmt.Errorf("handle ItemAddedToCart: %w", err)
			}

		case api.ItemRemovedFromCart:
			if err := p.handleItemRemovedFromCart(ctx, tx, event); err != nil {
				return fmt.Errorf("handle ItemRemovedFromCart: %w", err)
			}

		case api.CartCheckedOut:
			if err := p.handleCartCheckedOut(ctx, tx, event); err != nil {
				return fmt.Errorf("handle CartCheckedOut: %w", err)
			}
		}
	}

	// Update the last processed position
	_, err = tx.Exec(ctx, `
		UPDATE inventory_v2.last_processed_position
		SET position = $1
	`, maxPosition)
	if err != nil {
		return fmt.Errorf("update last_processed_position: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (p *Projection) handleItemAddedToCart(ctx context.Context, tx pgx.Tx, event es.Event) error {
	itemID, err := extractItemID(event.Data)
	if err != nil {
		return err
	}

	// Ensure the cart exists
	_, err = tx.Exec(ctx, `
		INSERT INTO inventory_v2.carts (cart_id, checked_out)
		VALUES ($1, FALSE)
		ON CONFLICT DO NOTHING
	`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("insert cart: %w", err)
	}

	// Update or insert the cart item
	result, err := tx.Exec(ctx, `
		UPDATE inventory_v2.cart_items
		SET quantity = quantity + 1
		WHERE cart_id = $1 AND item_id = $2
	`, event.AggregateID, itemID)
	if err != nil {
		return fmt.Errorf("update cart_items: %w", err)
	}

	// If no rows were updated, insert a new record
	if result.RowsAffected() == 0 {
		_, err := tx.Exec(ctx, `
			INSERT INTO inventory_v2.cart_items (cart_id, item_id, quantity)
			VALUES ($1, $2, 1)
		`, event.AggregateID, itemID)
		if err != nil {
			return fmt.Errorf("insert cart_items: %w", err)
		}
	}

	return nil
}

func (p *Projection) handleItemRemovedFromCart(ctx context.Context, tx pgx.Tx, event es.Event) error {
	itemID, err := extractItemID(event.Data)
	if err != nil {
		return err
	}

	// Update the quantity (decrement by 1)
	result, err := tx.Exec(ctx, `
		UPDATE inventory_v2.cart_items
		SET quantity = quantity - 1
		WHERE cart_id = $1 AND item_id = $2
	`, event.AggregateID, itemID)
	if err != nil {
		return fmt.Errorf("update cart_items: %w", err)
	}

	// If no rows were updated, insert with negative quantity
	if result.RowsAffected() == 0 {
		_, err := tx.Exec(ctx, `
			INSERT INTO inventory_v2.cart_items (cart_id, item_id, quantity)
			VALUES ($1, $2, -1)
		`, event.AggregateID, itemID)
		if err != nil {
			return fmt.Errorf("insert cart_items: %w", err)
		}
	}

	return nil
}

func (p *Projection) handleCartCheckedOut(ctx context.Context, tx pgx.Tx, event es.Event) error {
	// Update the cart to mark it as checked out
	_, err := tx.Exec(ctx, `
		UPDATE inventory_v2.carts
		SET checked_out = TRUE
		WHERE cart_id = $1
	`, event.AggregateID)
	if err != nil {
		return fmt.Errorf("update cart checked_out: %w", err)
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
