CREATE TABLE webhooks (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    target_url       TEXT        NOT NULL,
    subscribed_events TEXT[]     NOT NULL,
    secret           TEXT        NOT NULL,
    active           BOOLEAN     NOT NULL DEFAULT TRUE,
    max_attempts     INT         NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL
);