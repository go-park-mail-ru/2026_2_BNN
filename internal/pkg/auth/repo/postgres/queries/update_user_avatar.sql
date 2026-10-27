UPDATE users
SET avatar = $1
WHERE id = $2
RETURNING updated_at;
