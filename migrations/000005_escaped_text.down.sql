ALTER TABLE tags DROP CONSTRAINT tags_name_length;
ALTER TABLE tags ADD CONSTRAINT tags_name_length CHECK (char_length(name) BETWEEN 1 AND 30);

ALTER TABLE notes ALTER COLUMN title TYPE VARCHAR(255);