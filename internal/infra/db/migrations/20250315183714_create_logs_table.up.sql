CREATE TABLE logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    trace_id TEXT NULL,
    timestamp TIMESTAMP NOT NULL,
    level VARCHAR(10) NOT NULL,
    "service" TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB NULL
);