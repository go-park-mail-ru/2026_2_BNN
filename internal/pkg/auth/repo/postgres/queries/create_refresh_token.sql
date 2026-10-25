INSERT INTO refresh_tokens (token_hash, user_id, expires_at, created_at)
VALUES ($1, $2, $3, $4);