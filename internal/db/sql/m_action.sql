-- name: GetLastTimeMAction :one
SELECT cu.last_m_at
FROM chat_users cu
WHERE chat_tg_id = $1 AND user_tg_id = $2;

-- name: TryPerformMAction :one
UPDATE chat_users
SET
    m_action_count = m_action_count + 1,
    last_m_at = now()
WHERE chat_tg_id = $1
  AND user_tg_id = $2
  AND (last_m_at IS NULL OR last_m_at < NOW() - sqlc.arg(cooldown)::interval)
RETURNING m_action_count;