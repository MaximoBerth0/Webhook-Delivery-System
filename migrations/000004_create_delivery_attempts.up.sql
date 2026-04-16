CREATE TABLE delivery_attempts (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id     UUID        NOT NULL,
    attempt_number  INT         NOT NULL,
    status          TEXT        NOT NULL,
    response_code   INT,
    error_message   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_attempt_delivery FOREIGN KEY(delivery_id) REFERENCES deliveries(id)
);