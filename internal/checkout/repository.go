package checkout

import (
	"context"
	"es/internal/es"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CartRepository interface {
	New(context.Context, int) (*CartAggregate, error)
	Get(context.Context, int) (*CartAggregate, error)
	Save(context.Context, *CartAggregate) error
}

type PGCartRepository struct {
	pool *pgxpool.Pool
}

func NewPGCartRepository(pool *pgxpool.Pool) *PGCartRepository {
	return &PGCartRepository{
		pool: pool,
	}
}

func (r *PGCartRepository) New(ctx context.Context, cartID int) (*CartAggregate, error) {
	cart := NewCartAggregate(cartID)
	return cart, r.Save(ctx, cart)
}

func (r *PGCartRepository) Get(ctx context.Context, cartID int) (*CartAggregate, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	sql := `
        SELECT
        	position,
        	event_type,
         	aggregate_type,
          	aggregate_id,
         	at,
          	version_id,
           	data
        FROM events
        WHERE aggregate_id = $1 AND aggregate_type = $2
        ORDER BY version_id ASC`

	rows, err := conn.Query(ctx, sql, cartID, CartType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []es.Event{}
	for rows.Next() {
		var e es.Event

		if err := rows.Scan(
			&e.Position,
			&e.Type,
			&e.AggregateType,
			&e.AggregateID,
			&e.At,
			&e.VersionID,
			&e.Data,
		); err != nil {
			return nil, err
		}

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, nil
	}

	cart := &CartAggregate{}
	if err := cart.Apply(events...); err != nil {
		return nil, err
	}
	cart.Commit()
	return cart, nil
}

func (r *PGCartRepository) Save(ctx context.Context, cart *CartAggregate) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	for _, event := range cart.UncommittedEvents() {
		if err := event.Validate(); err != nil {
			return err
		}

		sql := `
			INSERT INTO events (aggregate_id, aggregate_type, event_type, at, version_id, data)
			VALUES ($1, $2, $3, $4, $5, $6)`

		_, err = conn.Exec(context.Background(), sql, event.AggregateID, event.AggregateType,
			event.Type, event.At, event.VersionID, event.Data)
		if err != nil {
			return err
		}
	}

	cart.Commit()
	return nil
}
