package es

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type ProjectionWriter interface {
	Name() string
	SubscribedEvents() []EventType
	ApplyMigration(context.Context) error
	LatestPosition(context.Context) (int64, error)
	Apply(context.Context, ...Event) error
}

type Subscription struct {
	writer          ProjectionWriter
	batchSize       int64
	refreshInterval time.Duration
}

func NewSubscription(
	writer ProjectionWriter,
	batchSize int64,
	refreshInterval time.Duration,
) *Subscription {
	return &Subscription{
		writer:          writer,
		batchSize:       batchSize,
		refreshInterval: refreshInterval,
	}
}

func (bp *Subscription) Listen(ctx context.Context, stream *EventStream) error {
	if err := bp.writer.ApplyMigration(ctx); err != nil {
		return fmt.Errorf("failed to apply migration: %w", err)
	}

	lastPosition, err := bp.writer.LatestPosition(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest position: %w", err)
	}

	ticker := time.NewTicker(bp.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf(
				"%s recieved shutdown signal, lastPosition=%d\n",
				bp.writer.Name(),
				lastPosition,
			)
			return nil
		case <-ticker.C:
			if err := bp.Refresh(ctx, stream, lastPosition); err != nil {
				return fmt.Errorf(
					"%s failed to refresh subscription: %w",
					bp.writer.Name(),
					err,
				)
			}
		}
	}
}

func (bp *Subscription) Refresh(
	ctx context.Context,
	stream *EventStream,
	lastPosition int64,
) error {
	subscribedEvents := bp.writer.SubscribedEvents()

	if len(subscribedEvents) == 0 {
		return errors.New("projection must subscribe to at least one event")
	}

	maxPosition, err := stream.GetMaxPosition(ctx)
	if err != nil {
		return fmt.Errorf("failed to get max position: %w", err)
	}

	for lastPosition < maxPosition {
		nextPosition := min(lastPosition+bp.batchSize+1, maxPosition)
		events, err := stream.GetEvents(
			ctx,
			lastPosition,
			nextPosition,
			subscribedEvents,
		)

		if err != nil {
			return fmt.Errorf("failed to get events: %w", err)
		}

		if len(events) == 0 {
			continue
		}

		if err := bp.writer.Apply(ctx, events...); err != nil {
			return fmt.Errorf("failed to apply events: %w", err)
		}

		lastPosition = nextPosition
	}

	fmt.Printf("%s position=%d\n", bp.writer.Name(), lastPosition)
	return nil
}
