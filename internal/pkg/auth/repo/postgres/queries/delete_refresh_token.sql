DELETE FROM refresh_tokens
WHERE token_hash = $1
RETURNING token_hash, user_id, expires_at, created_at;