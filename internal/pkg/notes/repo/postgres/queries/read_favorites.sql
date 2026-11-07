SELECT n.id, n.parent_note_id, n.created_by, n.title, n.header, n.icon, n.created_at, n.updated_at
FROM favorite f
JOIN notes n ON n.id = f.note_id
WHERE f.user_id = $1
ORDER BY n.updated_at DESC;