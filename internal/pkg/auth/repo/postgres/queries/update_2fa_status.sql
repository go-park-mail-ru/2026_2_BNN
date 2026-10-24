UPDATE users
SET is_2fa_enabled = $1
WHERE id = $2;