-- name: GetUsersForGameCursorBased :many
SELECT
    u.user_name,
    cu.user_tg_id,
    cu.last_message_at
FROM chat_users cu
JOIN users u ON cu.user_tg_id = u.tg_id
WHERE cu.chat_tg_id = $1
  AND cu.is_user_removed = false
  AND (cu.last_message_at, cu.user_tg_id) < (sqlc.arg(cursor), sqlc.arg(tiebreaker_tg_id)::bigint)
ORDER BY cu.last_message_at DESC, cu.user_tg_id DESC
LIMIT $2;

-- name: RecordGameLose :exec
UPDATE chat_users
SET
    loses = loses + 1
WHERE chat_tg_id = $1 AND user_tg_id = $2;

-- name: RecordGame :exec
UPDATE chat_users
SET
    games_played = games_played + 1
WHERE chat_tg_id = $1 AND user_tg_id = ANY(sqlc.arg(ids)::bigint[]);

-- name: StartGame :one
UPDATE chats
SET
    last_c_game_at = now()
WHERE tg_id = $1
  AND (last_c_game_at IS NULL OR last_c_game_at < NOW() - sqlc.arg(cooldown)::interval)
RETURNING last_c_game_at;

-- name: RemoveLostUsers :exec
UPDATE chat_users
SET
    is_user_removed = true
WHERE chat_tg_id = $1 AND user_tg_id = ANY(sqlc.arg(ids)::bigint[]);

-- name: GetChatMemberCount :one
SELECT COUNT(*)
FROM chat_users cu
WHERE cu.chat_tg_id = $1;

-- name: GetGameLastTime :one
SELECT c.last_c_game_at
FROM chats c
WHERE c.tg_id = $1;