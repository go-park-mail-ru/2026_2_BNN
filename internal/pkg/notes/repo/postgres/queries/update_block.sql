UPDATE blocks
SET content = $1, updated_at = $2
WHERE id = $3;