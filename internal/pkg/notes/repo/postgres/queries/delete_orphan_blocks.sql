DELETE FROM block b
WHERE b.id = ANY($1)
  AND NOT EXISTS (
      SELECT 1
      FROM note_block nb
      WHERE nb.block_id = b.id
  );
