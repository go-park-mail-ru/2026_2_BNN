SELECT id, login
FROM users
WHERE strpos(lower(login), lower($1)) > 0
ORDER BY login
LIMIT $2;