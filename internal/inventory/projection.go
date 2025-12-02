package inventory

import (
	"context"
	"errors"
	"es/internal/es"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Projection struct {
	pool *pgxpool.Pool
}

func NewProjection(pool *pgxpool.Pool) *Projection {
	return &Projection{pool: pool}
}

func (p *Projection) Apply(ctx context.Context, events ...es.Event) error {
	return errors.New("not implemented")
}

func (p *Projection) ApplyMigration(ctx context.Context) error {
	return errors.New("not implemented")
}

func (p *Projection) LatestPosition(ctx context.Context) (int64, error) {
	return 0, errors.New("not implemented")
}

func (p *Projection) SubscribedEvents() []es.EventType {
	return []es.EventType{}
}

func (p *Projection) Close(ctx context.Context) error {
	return errors.New("not implemented")
}

func (p *Projection) Subscribe(ctx context.Context) error {
	return errors.New("not implemented")
}
