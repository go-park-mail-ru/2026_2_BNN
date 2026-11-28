SELECT h.id, h.key_id, h.action, h.old_value, h.new_value,
       h.user_id, COALESCE(u.login, ''), h.created_at
FROM onlineconf_history h
LEFT JOIN users u ON u.id = h.user_id
WHERE h.key_id = $1
ORDER BY h.created_at DESC, h.id DESC;
