SELECT n.id, n.parent_note_id, n.created_by,
       COALESCE(n.title, ''), COALESCE(n.header, ''), COALESCE(n.icon, ''),
       n.version, n.created_at, n.updated_at
FROM note n
JOIN note_member m ON m.note_id = n.id
WHERE n.id = $1
  AND m.user_id = $2
  AND m.removed_at IS NULL;
