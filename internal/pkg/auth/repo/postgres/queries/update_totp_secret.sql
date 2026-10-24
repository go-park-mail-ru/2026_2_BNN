UPDATE users
SET totp_secret = $1
WHERE id = $2;