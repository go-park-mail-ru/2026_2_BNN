SELECT b.id
FROM block b
WHERE b.id = $1
  AND EXISTS (
      SELECT 1
      FROM note_block nb
      JOIN note_member m ON m.note_id = nb.note_id
      WHERE nb.block_id = b.id
        AND m.user_id = $2
        AND m.role = 'editor'
        AND m.removed_at IS NULL
  )
FOR NO KEY UPDATE;
