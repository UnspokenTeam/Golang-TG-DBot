-- name: GetLastTimeSAction :one
SELECT cu.last_s_at
FROM chat_users cu
WHERE chat_tg_id = $1 AND user_tg_id = $2;

-- name: TryPerformSAction :one
UPDATE chat_users
SET
    s_action_from_stranger_count = s_action_from_stranger_count + 1,
    last_s_at = now()
WHERE chat_tg_id = $1
  AND user_tg_id = $2
  AND (last_s_at IS NULL OR last_s_at < NOW() - sqlc.arg(cooldown)::interval)
RETURNING s_action_from_stranger_count;

-- name: ConfirmSAction :exec
UPDATE chat_users
SET
    s_action_count = s_action_count + 1
WHERE chat_tg_id = $1
  AND user_tg_id = $2;