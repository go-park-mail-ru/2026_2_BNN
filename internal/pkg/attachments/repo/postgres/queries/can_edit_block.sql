SELECT EXISTS (
    SELECT 1
    FROM note_block nb
    JOIN note_member m ON m.note_id = nb.note_id
    WHERE nb.block_id = $1
      AND m.user_id = $2
      AND m.role = 'editor'
      AND m.removed_at IS NULL
);
