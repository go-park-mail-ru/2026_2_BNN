SELECT user_id
FROM user_session
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > CURRENT_TIMESTAMP;
