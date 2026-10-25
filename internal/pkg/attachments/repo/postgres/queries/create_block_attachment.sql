INSERT INTO block_attachment (block_id, attachment_id, position)
SELECT $1, $2, COALESCE(MAX(position), -1) + 1
FROM block_attachment
WHERE block_id = $1;
