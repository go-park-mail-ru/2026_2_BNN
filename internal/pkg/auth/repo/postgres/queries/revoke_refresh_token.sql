UPDATE refresh_tokens
SET revoked_at = now()
WHERE token_hash = $1 AND revoked_at IS NULL
RETURNING id, token_hash, user_id, expires_at, created_at;
