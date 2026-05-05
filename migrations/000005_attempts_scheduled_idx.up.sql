-- search for pending attempts
CREATE INDEX idx_attempts_scheduled 
ON delivery_attempts(status, scheduled_for) 
WHERE status = 'pending';

-- search for delivery id
CREATE INDEX idx_attempts_delivery_id 
ON delivery_attempts(delivery_id);