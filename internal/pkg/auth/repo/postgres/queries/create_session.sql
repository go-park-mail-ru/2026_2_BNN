INSERT INTO user_session (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);