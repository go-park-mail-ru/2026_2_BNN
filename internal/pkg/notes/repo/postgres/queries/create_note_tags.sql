INSERT INTO note_tags (note_id, tag_id)
SELECT $1, unnest($2::uuid[]);