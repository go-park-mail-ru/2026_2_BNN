WITH inserted AS (
    INSERT INTO attachments (id, block_id, path)
    VALUES ($1, $2, $3)
    RETURNING block_id
),
updated_block AS (
    UPDATE blocks
    SET updated_at = NOW()
    WHERE id = (SELECT block_id FROM inserted)
    RETURNING note_id
)
UPDATE notes
SET updated_at = NOW()
WHERE id = (SELECT note_id FROM updated_block);