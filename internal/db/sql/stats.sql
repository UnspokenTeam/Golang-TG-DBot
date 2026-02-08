-- name: GetAllTimeStats :one
SELECT
    (SELECT COUNT(*) FROM users) AS total_users,
    (SELECT COUNT(*) FROM chats c WHERE c.is_active AND (c.type = 'supergroup' OR c.type = 'group')) AS total_chats;

-- name: GetAllAdminTimeStats :one
SELECT
    (SELECT COUNT(*)
     FROM users u
     WHERE EXISTS (
         SELECT 1 FROM chat_users cu
         WHERE cu.user_tg_id = u.tg_id
           AND cu.last_message_at > NOW() - INTERVAL '24 HOURS'
     )) AS today_active_users,

    (SELECT COUNT(*)
     FROM chats c
     WHERE c.is_active
       AND (c.type = 'supergroup' OR c.type = 'group')
       AND EXISTS (
         SELECT 1 FROM chat_users cu
         WHERE cu.chat_tg_id = c.tg_id
           AND cu.last_message_at > NOW() - INTERVAL '24 HOURS'
     )) AS today_active_chats,

    (SELECT COUNT(DISTINCT cu.user_tg_id)
     FROM chat_users cu
     WHERE cu.is_user_removed
       AND cu.last_message_at > NOW() - INTERVAL '24 HOURS'
    ) AS today_lazy_users,

    (SELECT COUNT(*)
     FROM chats c
     WHERE c.is_active = false
       AND (c.type = 'supergroup' OR c.type = 'group')
       AND EXISTS (
         SELECT 1 FROM chat_users cu
         WHERE cu.chat_tg_id = c.tg_id
           AND cu.last_message_at > NOW() - INTERVAL '24 HOURS'
     )) AS today_lazy_chats,

    (SELECT COUNT(*)
     FROM chats c
     WHERE c.created_at > NOW() - INTERVAL '24 HOURS') as today_new_chats,

    (SELECT COUNT(*)
     FROM users u
     WHERE u.created_at > NOW() - INTERVAL '24 HOURS') as today_new_users;



