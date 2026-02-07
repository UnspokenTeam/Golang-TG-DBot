-- name: GetLastTimeFAction :one
SELECT cu.last_s_at
FROM chat_users cu
WHERE chat_tg_id = $1 AND user_tg_id = $2;

-- name: TryPerformFAction :one
UPDATE chat_users
SET
    f_action_count = f_action_count + 1,
    last_s_at = now()
WHERE chat_tg_id = $1
  AND user_tg_id = $2
  AND (last_s_at IS NULL OR last_s_at < NOW() - sqlc.arg(cooldown)::interval)
RETURNING f_action_count;

-- name: ConfirmFAction :exec
UPDATE chat_users
SET
    f_action_from_stranger_count = f_action_from_stranger_count + 1
WHERE chat_tg_id = $1
  AND user_tg_id = $2;