INSERT INTO user_session (id, user_id, token_hash, expires_at)
VALUES ($1, $2, $3, $4);
