-- migrate:up
DROP FUNCTION init_chat_user_data;
CREATE FUNCTION init_chat_user_data(
    p_user_tg_id bigint,
    p_user_tag varchar(64),
    p_user_name varchar(128),
    p_user_lastname varchar(128),
    p_chat_tg_id bigint,
    p_chat_type varchar(64),
    p_chat_name varchar(512),
    p_member_count int
)
    RETURNS void AS $$
BEGIN

    INSERT INTO users (tg_id, user_tag, user_name, user_lastname)
    VALUES (p_user_tg_id, p_user_tag, p_user_name, p_user_lastname)
    ON CONFLICT (tg_id)
        DO UPDATE SET
                      user_tag = EXCLUDED.user_tag,
                      user_name = EXCLUDED.user_name,
                      user_lastname = EXCLUDED.user_lastname;

    INSERT INTO chats (tg_id, type, name, member_count, is_active)
    VALUES (p_chat_tg_id, p_chat_type, p_chat_name, p_member_count, true)
    ON CONFLICT (tg_id)
        DO UPDATE SET
                      type = EXCLUDED.type,
                      name = EXCLUDED.name,
                      member_count = EXCLUDED.member_count,
                      is_active = true;

    INSERT INTO chat_users (user_tg_id, chat_tg_id, is_user_removed)
    VALUES (p_user_tg_id, p_chat_tg_id, false)
    ON CONFLICT (chat_tg_id, user_tg_id)
        DO UPDATE SET
        is_user_removed = false;

END;
$$
    LANGUAGE plpgsql;


-- migrate:down
DROP FUNCTION init_chat_user_data;
CREATE FUNCTION init_chat_user_data(
    p_user_tg_id bigint,
    p_user_tag varchar (64),
    p_user_name varchar (128),
    p_user_lastname varchar (128),
    p_chat_tg_id bigint,
    p_chat_type varchar (64),
    p_chat_name varchar (512),
    p_member_count int
)
    RETURNS void AS $$
BEGIN

    INSERT INTO users (tg_id, user_tag, user_name, user_lastname)
    VALUES (p_user_tg_id, p_user_tag, p_user_name, p_user_lastname) ON CONFLICT (tg_id) DO NOTHING;

    INSERT INTO chats (tg_id, type, name, member_count)
    VALUES (p_chat_tg_id, p_chat_type, p_chat_name, p_member_count) ON CONFLICT (tg_id) DO NOTHING;

    INSERT INTO chat_users (user_tg_id, chat_tg_id)
    VALUES (p_user_tg_id, p_chat_tg_id) ON CONFLICT (chat_tg_id, user_tg_id) DO NOTHING;

END;
$$
    LANGUAGE plpgsql;