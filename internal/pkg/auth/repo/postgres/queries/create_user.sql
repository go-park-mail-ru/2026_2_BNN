INSERT INTO users (id, login, password_hash, avatar, totp_secret, is_2fa_enabled, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);