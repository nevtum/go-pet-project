package v2

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemCount struct {
	SoldCount   int `json:"sold"`
	StagedCount int `json:"reserved"`
}

type Result struct {
	ID    int       `json:"id"`
	Count ItemCount `json:"count"`
}

// ItemCountRepository defines the interface for retrieving item counts
type ItemCountRepository interface {
	GetItemCounts(ctx context.Context) ([]Result, error)
}

// PGItemCountRepository implements ItemCountRepository using PostgreSQL
type PGItemCountRepository struct {
	pool *pgxpool.Pool
}

// NewPGItemCountRepository creates a new instance of PGItemCountRepository
func NewPGItemCountRepository(pool *pgxpool.Pool) *PGItemCountRepository {
	return &PGItemCountRepository{
		pool: pool,
	}
}

// GetItemCounts retrieves sold and reserved counts for the given item IDs
func (r *PGItemCountRepository) GetItemCounts(ctx context.Context) ([]Result, error) {
	query := `
		SELECT
			ci.item_id,
			COALESCE(SUM(
				CASE
					WHEN c.checked_out = TRUE THEN ci.quantity
					ELSE 0
				END
			), 0) as sold_count,
			COALESCE(SUM(
				CASE
					WHEN c.checked_out = FALSE THEN ci.quantity
					ELSE 0
				END
			), 0) as reserved_count
		FROM inventory_v2.cart_items ci
		JOIN inventory_v2.carts c ON ci.cart_id = c.cart_id
		GROUP BY ci.item_id
	`

	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query item counts: %w", err)
	}
	defer rows.Close()

	results := make([]Result, 0)

	// Scan results
	for rows.Next() {
		var result Result
		if err := rows.Scan(&result.ID, &result.Count.SoldCount, &result.Count.StagedCount); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}
