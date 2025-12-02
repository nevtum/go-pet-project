CREATE TABLE IF NOT EXISTS events (
    position SERIAL PRIMARY KEY,
    aggregate_id INTEGER NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    at TIMESTAMP NOT NULL,
    version_id INTEGER NOT NULL,
    data JSONB
);

CREATE INDEX IF NOT EXISTS idx_events_type ON events (event_type);
CREATE INDEX IF NOT EXISTS idx_events_at ON events (at);
CREATE INDEX IF NOT EXISTS idx_events_aggregate_type ON events (aggregate_id, aggregate_type);
CREATE INDEX IF NOT EXISTS idx_events_aggregate_at ON events (aggregate_id, at);
