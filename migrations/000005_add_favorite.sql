CREATE TABLE IF NOT EXISTS favorite (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    note_id UUID NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, note_id)
);