CREATE TABLE IF NOT EXISTS deliveries (
    id              TEXT PRIMARY KEY,
    notification_id TEXT NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    channel         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING',
    attempts        INT  NOT NULL DEFAULT 0,
    last_error      TEXT NOT NULL DEFAULT '',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS deliveries_notification_id_idx ON deliveries (notification_id);
CREATE INDEX IF NOT EXISTS deliveries_status_idx ON deliveries (status);
