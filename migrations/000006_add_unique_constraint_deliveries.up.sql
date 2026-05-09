ALTER TABLE deliveries 
ADD CONSTRAINT unique_delivery_per_event_webhook 
UNIQUE (event_id, webhook_id);