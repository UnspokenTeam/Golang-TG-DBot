-- migrate:up
CREATE TABLE "chats"
(
    "id"                  bigserial PRIMARY KEY,
    "tg_id"               bigint UNIQUE NOT NULL,
    "type"                varchar(64)   NOT NULL DEFAULT 'GROUP',
    "name"                varchar(512)  NOT NULL,
    "last_c_game_at"      timestamp              DEFAULT null,
    "last_sys_message_at" timestamp              DEFAULT null,
    "last_message_at"     timestamp     NOT NULL DEFAULT (now()),
    "is_active"           bool          NOT NULL DEFAULT true,
    "member_count"        int           NOT NULL DEFAULT 1,
    "created_at"          timestamp     NOT NULL DEFAULT (now()),
    "updated_at"          timestamp     NOT NULL DEFAULT (now())
);

CREATE TABLE "chat_users"
(
    "user_tg_id"                   bigint         NOT NULL,
    "chat_tg_id"                   bigint         NOT NULL,
    "is_user_removed"              bool           NOT NULL DEFAULT false,
    "last_message_at"              timestamp      NOT NULL DEFAULT (now()),
    "m_action_count"               int            NOT NULL DEFAULT 0,
    "last_m_at"                    timestamp               DEFAULT null,
    "d_length"                     numeric(10, 1) NOT NULL DEFAULT 0,
    "last_grow_at"                 timestamp               DEFAULT null,
    "f_action_count"               int            NOT NULL DEFAULT 0,
    "last_f_at"                    timestamp               DEFAULT null,
    "f_action_from_stranger_count" int            NOT NULL DEFAULT 0,
    "s_action_count"               int            NOT NULL DEFAULT 0,
    "last_s_at"                    timestamp               DEFAULT null,
    "s_action_from_stranger_count" int            NOT NULL DEFAULT 0,
    "games_played"                 int            NOT NULL DEFAULT 0,
    "loses"                        int            NOT NULL DEFAULT 0,
    "created_at"                   timestamp      NOT NULL DEFAULT (now()),
    PRIMARY KEY ("chat_tg_id", "user_tg_id")
);

CREATE TABLE "users"
(
    "id"            bigserial PRIMARY KEY,
    "tg_id"         bigint UNIQUE NOT NULL,
    "user_tag"      varchar(64),
    "user_name"     varchar(128)  NOT NULL,
    "user_lastname" varchar(128),
    "user_role"     varchar(64)   NOT NULL DEFAULT 'USER',
    "created_at"    timestamp     NOT NULL DEFAULT (now()),
    "updated_at"    timestamp     NOT NULL DEFAULT (now())
);

CREATE TABLE "actions"
(
    "id"          bigserial PRIMARY KEY,
    "is_yourself" bool      NOT NULL DEFAULT (true),
    "chat_tg_id"     bigint    NOT NULL,
    "user_tg_id"     bigint    NOT NULL,
    "action"      text      NOT NULL,
    "created_at"  timestamp NOT NULL DEFAULT (now())
);

ALTER TABLE "chat_users"
    ADD FOREIGN KEY ("chat_tg_id") REFERENCES "chats" ("tg_id");

ALTER TABLE "actions"
    ADD FOREIGN KEY ("chat_tg_id") REFERENCES "chats" ("tg_id");

ALTER TABLE "chat_users"
    ADD FOREIGN KEY ("user_tg_id") REFERENCES "users" ("tg_id");

ALTER TABLE "actions"
    ADD FOREIGN KEY ("user_tg_id") REFERENCES "users" ("tg_id");

CREATE INDEX "ix_chat_users_chat_tg_id" ON "chat_users" ("chat_tg_id");

CREATE INDEX "ix_chat_users_user_tg_id" ON "chat_users" ("user_tg_id");

CREATE INDEX ix_chats_member_count_active ON "chats" ("member_count") WHERE is_active;

CREATE INDEX ix_chat_users_active_messages
    ON "chat_users" ("chat_tg_id", "last_message_at" DESC) WHERE is_user_removed = false;

CREATE INDEX ix_chat_users_d_length_active_desc
    ON "chat_users" ("d_length" DESC);

CREATE INDEX ix_actions_yourself_created_at
    ON "actions" ("is_yourself", "created_at" DESC);

CREATE
OR REPLACE FUNCTION update_last_message_at()
RETURNS TRIGGER AS $$
BEGIN

IF pg_trigger_depth() > 1 THEN
    RETURN NEW;
END IF;

UPDATE chats
SET last_message_at = NOW()
WHERE tg_id = NEW.chat_tg_id;

UPDATE chat_users
SET last_message_at = NOW()
WHERE chat_tg_id = NEW.chat_tg_id
  AND user_tg_id = NEW.user_tg_id;

RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER trigger_actions_update_last_message
    AFTER INSERT OR
UPDATE ON actions
    FOR EACH ROW
    EXECUTE FUNCTION update_last_message_at();

CREATE TRIGGER trigger_chat_users_update_last_message
    AFTER INSERT OR
UPDATE ON chat_users
    FOR EACH ROW
    EXECUTE FUNCTION update_last_message_at();

-- migrate:down
DROP TRIGGER trigger_chat_users_update_last_message ON chat_users;
DROP TRIGGER trigger_actions_update_last_message ON actions;

DROP FUNCTION update_last_message_at();

DROP INDEX ix_actions_yourself_created_at;
DROP INDEX ix_chat_users_d_length_active_desc;
DROP INDEX ix_chat_users_active_messages;
DROP INDEX ix_chats_member_count_active;
DROP INDEX ix_chat_users_user_tg_id;
DROP INDEX ix_chat_users_chat_tg_id;

DROP TABLE actions CASCADE;
DROP TABLE chat_users CASCADE;
DROP TABLE users CASCADE;
DROP TABLE chats CASCADE;

