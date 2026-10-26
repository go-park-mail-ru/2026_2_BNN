SELECT id, created_by, name, color, created_at, updated_at
FROM tags
WHERE id = $1;