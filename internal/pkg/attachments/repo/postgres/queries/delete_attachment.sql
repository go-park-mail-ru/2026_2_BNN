WITH deleted AS (
    DELETE FROM attachments
    WHERE id = $1
    RETURNING block_id
),
updated_block AS (
    UPDATE blocks
    SET updated_at = NOW()
    WHERE id = (SELECT block_id FROM deleted)
    RETURNING note_id
)
UPDATE notes
SET updated_at = NOW()
WHERE id = (SELECT note_id FROM updated_block);