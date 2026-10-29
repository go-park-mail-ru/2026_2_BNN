SELECT b.id, b.content, b.created_at, b.updated_at
FROM note_block nb
JOIN block b ON b.id = nb.block_id
WHERE nb.note_id = $1
ORDER BY nb.position;
