SELECT id, parent_note_id, created_by, title, header, icon, created_at, updated_at
FROM notes
WHERE created_by = $1
ORDER BY updated_at DESC, id
LIMIT $2 OFFSET $3;