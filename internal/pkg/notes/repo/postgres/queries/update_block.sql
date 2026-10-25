WITH changed AS (
    UPDATE block
    SET content = $2
    WHERE id = $1
      AND content IS DISTINCT FROM $2
    RETURNING id, content
)
INSERT INTO block_version (block_id, version, content, created_by)
SELECT c.id,
       (SELECT COALESCE(MAX(v.version), 0) + 1
        FROM block_version v
        WHERE v.block_id = c.id),
       c.content,
       $3
FROM changed c;
