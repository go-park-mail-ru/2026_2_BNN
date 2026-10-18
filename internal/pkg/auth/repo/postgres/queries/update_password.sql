UPDATE users
SET password_hash = $1, updated_at = now()
WHERE id = $2;