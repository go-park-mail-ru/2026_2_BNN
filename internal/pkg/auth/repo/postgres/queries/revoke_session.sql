UPDATE user_session
SET revoked_at = CURRENT_TIMESTAMP
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > CURRENT_TIMESTAMP
RETURNING user_id;