SELECT id, login, password_hash, avatar, created_at, updated_at
FROM users
WHERE id = $1;