CREATE TABLE deliveries (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    UUID        NOT NULL,
    webhook_id  UUID        NOT NULL,
    status      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    -- Constraints 
    CONSTRAINT fk_delivery_event   FOREIGN KEY (event_id)   REFERENCES events(id),
    CONSTRAINT fk_delivery_webhook FOREIGN KEY (webhook_id) REFERENCES webhooks(id),
    CONSTRAINT unique_delivery_per_event_webhook UNIQUE (event_id, webhook_id)
);