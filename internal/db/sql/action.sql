-- name: GetRandomActionForStrangerFromNewest :one
SELECT *
FROM (
    SELECT id, action
    FROM actions
    WHERE is_yourself = false
    ORDER BY created_at DESC
    LIMIT 1000
) AS newest
ORDER BY RANDOM()
LIMIT 1;

-- name: GetYourselfRandomActionFromNewest :one
SELECT *
FROM (
         SELECT id, action
         FROM actions
         WHERE is_yourself
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
