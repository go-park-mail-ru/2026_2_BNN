SELECT n.version
FROM note n
JOIN note_member m ON m.note_id = n.id
WHERE n.id = $1
  AND m.user_id = $2
  AND m.role = 'editor'
  AND m.removed_at IS NULL;
