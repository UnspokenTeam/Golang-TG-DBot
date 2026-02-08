-- name: GetRandomActionFromNewest :one
SELECT *
FROM (
    SELECT id, action
    FROM actions
    WHERE is_yourself = $1
    ORDER BY created_at DESC
    LIMIT 1000
) AS newest
ORDER BY RANDOM()
LIMIT 1;

-- name: UpdateLastMessageAt :exec
UPDATE chat_users
SET last_message_at = NOW()
WHERE chat_tg_id = $1
  AND user_tg_id = $2;

-- name: InsertNewAction :exec
INSERT INTO actions (is_yourself, chat_tg_id, user_tg_id, action)
VALUES ($1, $2, $3, $4);
