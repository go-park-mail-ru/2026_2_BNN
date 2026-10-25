SELECT id
FROM note
WHERE id = $1
  AND created_by = $2
FOR UPDATE;
