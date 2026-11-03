UPDATE tags
SET name = $1, color = $2, updated_at = $3
WHERE id = $4;