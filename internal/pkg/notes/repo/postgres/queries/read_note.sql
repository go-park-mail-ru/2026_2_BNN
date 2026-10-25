SELECT id, parent_note_id, created_by, title, header, icon, created_at, updated_at
FROM notes
WHERE id = $1;