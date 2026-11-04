SELECT id, login, password_hash, avatar, COALESCE(email, ''), is_external, COALESCE(totp_secret, ''), is_2fa_enabled, created_at, updated_at
FROM users
WHERE login = $1;