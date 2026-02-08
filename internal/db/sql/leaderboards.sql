-- name: GetChatLeaderBoards :many
WITH top AS (
    SELECT
        user_tg_id,
        d_length,
        m_action_count,
        f_action_count,
        f_action_from_stranger_count,
        s_action_count,
        s_action_from_stranger_count,
        loses
    FROM chat_users
    WHERE chat_tg_id = $1
    ORDER BY d_length DESC
    LIMIT 10
)
SELECT
    u.user_name,
    u.user_tag,
    top.*
FROM top
JOIN users u ON u.tg_id = top.user_tg_id
ORDER BY top.d_length DESC;

-- name: GetGlobalLeaderBoards :many
SELECT
    u.user_name,
    u.tg_id,
    u.user_tag,
    cu.*
FROM users u
CROSS JOIN LATERAL (
    SELECT
        d_length,
        m_action_count,
        f_action_count,
        f_action_from_stranger_count,
        s_action_count,
        s_action_from_stranger_count,
        loses
    FROM chat_users
    WHERE user_tg_id = u.tg_id
    ORDER BY d_length DESC
    LIMIT 1
    ) cu
ORDER BY cu.d_length DESC
LIMIT 10;
