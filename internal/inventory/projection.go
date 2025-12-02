package inventory

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

func (p *Projection) SubscribedEvents() []es.EventType {
	return []es.EventType{
		api.ItemAddedToCart,
		api.ItemRemovedFromCart,
	}
}

func (p *Projection) ApplyMigration(ctx context.Context) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS inventory_projection;`)

	// Create cart_items table
	tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS inventory_projection.cart_items (
			cart_id INTEGER NOT NULL,
			item_id INTEGER NOT NULL,
			quantity INTEGER NOT NULL,
			checked_out BOOLEAN NOT NULL,
			PRIMARY KEY (cart_id, item_id)
		);

		-- Index to help with querying total quantity of sold items by item_id
		CREATE INDEX IF NOT EXISTS idx_inventory_item_sold
		ON inventory_projection.cart_items (item_id, checked_out)
		WHERE checked_out = true;
	`)

	// Create last_processed_position table
	tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS inventory_projection.last_processed_position (
			position INTEGER NOT NULL,
			CONSTRAINT single_row CHECK (position >= 0)
		);

		-- Ensure only one row exists
		INSERT INTO inventory_projection.last_processed_position (position)
		SELECT 0
		WHERE NOT EXISTS (
			SELECT 1 FROM inventory_projection.last_processed_position
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
		FROM inventory_projection.last_processed_position
		LIMIT 1
	`).Scan(&position)
	if err != nil {
		return 0, fmt.Errorf("read latest position: %w", err)
	}
	return position, nil
}

func (p *Projection) Apply(ctx context.Context, events ...es.Event) error {
	return errors.New("not implemented")
}
