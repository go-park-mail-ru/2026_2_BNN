SELECT block_id
FROM note_block
WHERE note_id = ANY($1);
