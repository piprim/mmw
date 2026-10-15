-- +goose Up
-- +goose StatementBegin

CREATE SCHEMA IF NOT EXISTS [[.Name]];

-- Transactional outbox: a command writes its events here in the same
-- transaction as its state change; the relay publishes them to the bus.
CREATE TABLE IF NOT EXISTS [[.Name]].event (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_[[.Name]]_event_unpublished ON [[.Name]].event(occurred_at ASC) WHERE published_at IS NULL;
CREATE INDEX idx_[[.Name]]_event_published ON [[.Name]].event(published_at) WHERE published_at IS NOT NULL;

COMMENT ON TABLE [[.Name]].event IS 'Transactional outbox for event publishing via workers';
COMMENT ON COLUMN [[.Name]].event.published_at IS 'NULL indicates unpublished event, non-NULL means published';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS [[.Name]].idx_[[.Name]]_event_published;
DROP INDEX IF EXISTS [[.Name]].idx_[[.Name]]_event_unpublished;
DROP TABLE IF EXISTS [[.Name]].event;

-- +goose StatementEnd
