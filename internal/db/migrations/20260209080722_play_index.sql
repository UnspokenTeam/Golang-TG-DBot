-- migrate:up
DROP INDEX ix_chat_users_active_messages;
CREATE INDEX ix_chat_users_active_messages_seek
    ON chat_users (chat_tg_id, last_message_at DESC, user_tg_id DESC)
    WHERE is_user_removed = false;


-- migrate:down
DROP INDEX ix_chat_users_active_messages_seek;
CREATE INDEX ix_chat_users_active_messages
    ON "chat_users" ("chat_tg_id", "last_message_at" DESC) WHERE is_user_removed = false;