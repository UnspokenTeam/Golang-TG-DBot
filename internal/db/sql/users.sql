-- name: InitChatUserData :exec
SELECT init_chat_user_data($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetUserByTgId :one
SELECT *
FROM users u
WHERE u.tg_id = $1;

-- name: SetUserRoleByTgId :exec
UPDATE users u
SET user_role = $1
WHERE u.tg_id = $2;

-- name: GetUserStatsByTgId :one
SELECT
    u.user_name,
    c.name,
    cu.d_length,
    cu.m_action_count,
    cu.f_action_count,
    cu.s_action_count,
    cu.loses
FROM chat_users cu
JOIN chats c on cu.chat_tg_id = c.tg_id
JOIN users u on cu.user_tg_id = u.tg_id
WHERE cu.chat_tg_id = $1 AND cu.user_tg_id = $2;
