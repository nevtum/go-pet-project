package es

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EventStream struct {
	pool      *pgxpool.Pool
	batchSize int64
}

func NewEventStream(pool *pgxpool.Pool, batchSize int64) *EventStream {
	return &EventStream{
		pool:      pool,
		batchSize: batchSize,
	}
}

func (s *EventStream) Subscribe(ctx context.Context, projection ProjectionWriter) error {
	if err := projection.ApplyMigration(ctx); err != nil {
		return err
	}

	lastPosition, err := projection.LatestPosition(ctx)

	if err != nil {
		return err
	}

	for {
		events, err := s.getEvents(
			ctx,
			lastPosition,
			lastPosition+s.batchSize,
			projection.SubscribedEvents(),
		)

		if err != nil {
			return err
		}

		if len(events) == 0 {
			break
		}

		if err := projection.Apply(ctx, events...); err != nil {
			return err
		}

		lastPosition = events[len(events)-1].Position
	}

	return nil
}

func (s *EventStream) getEvents(ctx context.Context, startPos, endPos int64, eventTypes []EventType) ([]Event, error) {
	panic("not implemented")
}
