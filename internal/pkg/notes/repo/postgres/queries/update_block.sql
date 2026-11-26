WITH updated_block AS (
    UPDATE blocks
    SET content = $3, updated_at = $4
    WHERE id = $1 AND note_id = $2
    RETURNING note_id
)
UPDATE notes
SET updated_at = $4
WHERE id = (SELECT note_id FROM updated_block);