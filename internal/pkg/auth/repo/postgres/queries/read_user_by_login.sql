SELECT id, login, password_hash, avatar, totp_secret, is_2fa_enabled, created_at, updated_at
FROM users
WHERE login = $1;