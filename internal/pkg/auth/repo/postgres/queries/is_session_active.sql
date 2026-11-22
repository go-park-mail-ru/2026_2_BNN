SELECT EXISTS (
    SELECT 1
    FROM refresh_tokens
    WHERE id = $1 AND revoked_at IS NULL AND expires_at > now()
);