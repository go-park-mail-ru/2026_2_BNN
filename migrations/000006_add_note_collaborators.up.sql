CREATE TABLE IF NOT EXISTS note_collaborators (
    note_id    UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       VARCHAR(20) NOT NULL DEFAULT 'editor',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    
    PRIMARY KEY (note_id, user_id),
    
    CONSTRAINT valid_role CHECK (role IN ('editor', 'viewer'))
);

CREATE INDEX IF NOT EXISTS note_collaborators_user_id_idx ON note_collaborators(user_id);