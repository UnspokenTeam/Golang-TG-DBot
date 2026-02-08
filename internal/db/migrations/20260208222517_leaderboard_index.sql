-- migrate:up
CREATE INDEX ix_chat_users_chat_dlen_desc
    ON chat_users (chat_tg_id, d_length DESC);


-- migrate:down
DROP INDEX ix_chat_users_chat_dlen_desc;
