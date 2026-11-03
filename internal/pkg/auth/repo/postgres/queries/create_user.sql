INSERT INTO users (id, login, password_hash, avatar, email, is_external, totp_secret, is_2fa_enabled, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);