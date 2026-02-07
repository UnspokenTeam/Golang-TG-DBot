-- name: GrowD :one
UPDATE chat_users
SET
    d_length = GREATEST(d_length + $1, 0),
    last_grow_at = now()
WHERE chat_tg_id = $2
  AND user_tg_id = $3
  AND (last_grow_at IS NULL OR last_grow_at < NOW() - sqlc.arg(cooldown)::interval)
RETURNING d_length;

-- name: GetLastTimeDAction :one
SELECT cu.last_grow_at
FROM chat_users cu
WHERE chat_tg_id = $1 AND user_tg_id = $2;