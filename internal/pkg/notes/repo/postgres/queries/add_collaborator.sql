INSERT INTO note_collaborators (note_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (note_id, user_id)
DO UPDATE SET role = EXCLUDED.role;