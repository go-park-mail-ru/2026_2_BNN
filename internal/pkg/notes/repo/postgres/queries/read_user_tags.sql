SELECT id, created_by, name, color, created_at, updated_at
FROM tags
WHERE created_by = $1 AND id = ANY($2)
ORDER BY name;