SELECT id, token_hash, user_id, expires_at, created_at, revoked_at
FROM refresh_tokens
WHERE token_hash = $1;
