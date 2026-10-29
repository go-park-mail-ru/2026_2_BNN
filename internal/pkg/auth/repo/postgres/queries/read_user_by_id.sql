SELECT id, login, password_hash, COALESCE(avatar, ''), created_at, updated_at
FROM users
WHERE id = $1;