UPDATE users
SET avatar = $1, updated_at = $2
WHERE id = $3;