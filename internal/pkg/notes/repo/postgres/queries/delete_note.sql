DELETE FROM note
WHERE id = $1
  AND created_by = $2;
