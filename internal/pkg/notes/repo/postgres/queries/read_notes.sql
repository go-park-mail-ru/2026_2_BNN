SELECT DISTINCT n.id, n.parent_note_id, n.created_by, n.title, n.header, n.icon, n.created_at, n.updated_at
FROM notes n
LEFT JOIN note_collaborators nc ON nc.note_id = n.id AND nc.user_id = $1
WHERE n.created_by = $1 OR nc.user_id = $1
ORDER BY n.updated_at DESC
LIMIT $2 OFFSET $3;