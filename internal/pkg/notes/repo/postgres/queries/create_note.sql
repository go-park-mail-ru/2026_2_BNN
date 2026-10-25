INSERT INTO note (id, parent_note_id, created_by, title, header, icon, created_at, updated_at)
VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8);
