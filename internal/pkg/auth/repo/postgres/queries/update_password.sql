UPDATE users
SET password_hash = $1
WHERE id = $2
  AND password_hash = $3;