package es

import (
	"context"
)

type ProjectionWriter interface {
	SubscribedEvents() []EventType
	ApplyMigration(context.Context) error
	LatestPosition(context.Context) (int64, error)
	Apply(context.Context, ...Event) error
}
