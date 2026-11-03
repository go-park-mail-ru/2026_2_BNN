SELECT t.id, t.created_by, t.name, t.color, t.created_at, t.updated_at
FROM note_tags nt
JOIN tags t ON t.id = nt.tag_id
WHERE nt.note_id = $1
ORDER BY t.name;