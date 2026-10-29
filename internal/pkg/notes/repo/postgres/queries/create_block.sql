WITH inserted AS (
    INSERT INTO block (id, content)
    VALUES ($1, $2)
    RETURNING id, content
)
INSERT INTO block_version (block_id, version, content, created_by)
SELECT id, 1, content, $3
FROM inserted;
