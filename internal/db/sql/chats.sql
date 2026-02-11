-- name: GetChatCountForBroadcast :one
SELECT COUNT(*)
FROM chats
WHERE (type = 'supergroup' OR type = 'private' OR type = 'group')
AND is_active;

-- name: UpdateChatStatusToDead :exec
UPDATE chats
SET
    is_active = false
WHERE tg_id = ANY(sqlc.arg(dead_ids)::bigint[]);

-- name: UpdateLastSysTimestamp :exec
UPDATE chats
SET
    last_sys_message_at = now()
WHERE tg_id = ANY(sqlc.arg(alive_chats)::bigint[]);

-- name: GetChatsForBroadcastCursorBased :many
SELECT
    c.id,
    c.tg_id
FROM chats c
WHERE (type = 'supergroup' OR type = 'private' OR type = 'group')
  AND is_active = true
  AND id > sqlc.arg(cursor_id)::bigint
ORDER BY id
LIMIT sqlc.arg(page_size)::int;
