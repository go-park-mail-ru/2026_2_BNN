-- header и icon: NULL — оставить как есть, пустая строка — очистить.
UPDATE note n
SET title   = NULLIF($1, ''),
    header  = CASE WHEN $2::text IS NULL THEN n.header ELSE NULLIF($2, '') END,
    icon    = CASE WHEN $3::text IS NULL THEN n.icon ELSE NULLIF($3, '') END,
    version = n.version + 1
WHERE n.id = $4
  AND n.version = $5
  AND EXISTS (
      SELECT 1
      FROM note_member m
      WHERE m.note_id = n.id
        AND m.user_id = $6
        AND m.role = 'editor'
        AND m.removed_at IS NULL
  )
RETURNING n.id, n.parent_note_id, n.created_by,
          COALESCE(n.title, ''), COALESCE(n.header, ''), COALESCE(n.icon, ''),
          n.version, n.created_at, n.updated_at;
