package es

import (
	"context"
	"errors"
	"fmt"
)

type ProjectionWriter interface {
	SubscribedEvents() []EventType
	ApplyMigration(context.Context) error
	LatestPosition(context.Context) (int64, error)
	Apply(context.Context, ...Event) error
}

type Subscription struct {
	writer    ProjectionWriter
	batchSize int64
}

func NewSubscription(writer ProjectionWriter, batchSize int64) *Subscription {
	return &Subscription{
		writer:    writer,
		batchSize: batchSize,
	}
}

func (bp *Subscription) Listen(ctx context.Context, stream *EventStream) error {
	subscribedEvents := bp.writer.SubscribedEvents()

	if len(subscribedEvents) == 0 {
		return errors.New("projection must subscribe to at least one event")
	}

	if err := bp.writer.ApplyMigration(ctx); err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	lastPosition, err := bp.writer.LatestPosition(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest position: %w", err)
	}

	maxPosition, err := stream.GetMaxPosition(ctx)
	if err != nil {
		return fmt.Errorf("failed to get max position: %w", err)
	}

	for lastPosition < maxPosition {
		events, err := stream.GetEvents(
			ctx,
			lastPosition,
			lastPosition+bp.batchSize,
			subscribedEvents,
		)

		if err != nil {
			return fmt.Errorf("failed to get events: %w", err)
		}

		lastPosition += bp.batchSize + 1

		if len(events) == 0 {
			continue
		}

		if err := bp.writer.Apply(ctx, events...); err != nil {
			return fmt.Errorf("failed to apply events: %w", err)
		}
	}

	fmt.Println("no more events to process")
	return nil
}
