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