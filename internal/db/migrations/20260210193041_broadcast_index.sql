-- migrate:up
CREATE INDEX idx_chats_broadcast_pagination
    ON chats (type, is_active, id)
    WHERE is_active = true;

-- migrate:down
DROP INDEX idx_chats_broadcast_pagination
