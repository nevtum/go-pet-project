package es

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EventStream struct {
	pool *pgxpool.Pool
}

func NewEventStream(pool *pgxpool.Pool) *EventStream {
	return &EventStream{
		pool: pool,
	}
}

func (s *EventStream) GetMaxPosition(ctx context.Context) (int64, error) {
	query := `
		SELECT MAX(position)
		FROM events`

	var maxPosition int64
	err := s.pool.QueryRow(ctx, query).Scan(&maxPosition)
	if err != nil {
		return 0, err
	}

	return maxPosition, nil
}

func (s *EventStream) GetEvents(ctx context.Context, startPos, endPos int64, eventTypes []EventType) ([]Event, error) {
	query := `
		SELECT
			position,
			aggregate_id,
			aggregate_type,
			event_type,
			at,
			version_id,
			data
		FROM events
		WHERE position >= $1 AND position <= $2
		AND event_type = ANY($3)
		ORDER BY position ASC`

	rows, err := s.pool.Query(ctx, query, startPos, endPos, eventTypes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var dataJSON []byte

		err := rows.Scan(
			&e.Position,
			&e.AggregateID,
			&e.AggregateType,
			&e.Type,
			&e.At,
			&e.VersionID,
			&dataJSON,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON data if present
		if dataJSON != nil {
			if err := json.Unmarshal(dataJSON, &e.Data); err != nil {
				return nil, err
			}
		}

		events = append(events, e)
	}

	// Check for any errors during iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}
