CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id TEXT NOT NULL,
    type TEXT NOT NULL,
    payload BYTEA NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ NULL,
    correlation_id TEXT NULL
);

CREATE INDEX idx_outbox_sent_at ON outbox (sent_at);
CREATE INDEX idx_outbox_aggregate_id ON outbox (aggregate_id);



