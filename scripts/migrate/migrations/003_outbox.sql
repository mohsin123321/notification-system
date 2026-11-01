CREATE TABLE IF NOT EXISTS outbox (
    id           TEXT PRIMARY KEY,
    routing_key  TEXT        NOT NULL,
    body         BYTEA       NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS outbox_pending_idx ON outbox (published_at) WHERE published_at IS NULL;
