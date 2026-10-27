-- Вложенную заметку может создать только editor родителя.
-- FOR SHARE не даёт снять членство, пока заметка создаётся.
INSERT INTO note (id, parent_note_id, created_by, title, header, icon, created_at, updated_at)
SELECT $1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8
WHERE $2::uuid IS NULL
   OR EXISTS (
       SELECT 1
       FROM note_member m
       WHERE m.note_id = $2
         AND m.user_id = $3
         AND m.role = 'editor'
         AND m.removed_at IS NULL
       FOR SHARE
   );
