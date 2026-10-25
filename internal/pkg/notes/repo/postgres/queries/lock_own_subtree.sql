-- Заметка владельца и все её потомки. Блокируются в порядке id,
-- чтобы два удаления пересекающихся поддеревьев не взаимоблокировались.
WITH RECURSIVE subtree AS (
    SELECT id
    FROM note
    WHERE id = $1
      AND created_by = $2
    UNION ALL
    SELECT n.id
    FROM note n
    JOIN subtree s ON n.parent_note_id = s.id
)
SELECT n.id
FROM note n
JOIN subtree s ON s.id = n.id
ORDER BY n.id
FOR UPDATE OF n;
