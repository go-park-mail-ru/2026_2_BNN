SELECT b.id, b.content, b.created_at, b.updated_at
FROM notes n
CROSS JOIN unnest(n.block_ids) WITH ORDINALITY AS o(block_id, pos)
JOIN blocks b ON b.id = o.block_id
WHERE n.id = $1
ORDER BY o.pos;