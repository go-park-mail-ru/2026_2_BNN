SELECT a.path
FROM attachment a
WHERE a.id = $1
  AND EXISTS (
      SELECT 1
      FROM block_attachment ba
      JOIN note_block nb ON nb.block_id = ba.block_id
      JOIN note_member m ON m.note_id = nb.note_id
      WHERE ba.attachment_id = a.id
        AND m.user_id = $2
        AND m.removed_at IS NULL
  );
