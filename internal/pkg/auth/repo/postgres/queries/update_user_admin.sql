UPDATE users
SET is_admin = $1
WHERE id = $2;
