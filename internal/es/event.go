package es

import (
	"errors"
	"time"
)

type EventType string
type AggregateType string

type Event struct {
	Position      int64
	Type          EventType
	At            time.Time
	VersionID     int
	AggregateType AggregateType
	AggregateID   int
	Data          any
}

func (e Event) Validate() error {
	if e.AggregateID <= 0 {
		return errors.New("invalid aggregate ID")
	}
	if e.AggregateType == "" {
		return errors.New("aggregate type must not be empty")
	}
	if e.Type == "" {
		return errors.New("event type must not be empty")
	}
	if e.VersionID <= 0 {
		return errors.New("invalid version ID")
	}
	return nil
}
