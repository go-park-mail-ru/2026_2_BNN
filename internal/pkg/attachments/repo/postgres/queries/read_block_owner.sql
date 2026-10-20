SELECT n.created_by
FROM blocks b
JOIN notes n ON n.id = b.note_id
WHERE b.id = $1;