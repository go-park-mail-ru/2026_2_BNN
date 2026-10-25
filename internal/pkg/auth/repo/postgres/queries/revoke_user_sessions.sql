UPDATE user_session
SET revoked_at = CURRENT_TIMESTAMP
WHERE user_id = $1
  AND revoked_at IS NULL;