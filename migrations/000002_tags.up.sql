CREATE TABLE tags (
    id         UUID PRIMARY KEY,
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    color      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT tags_name_length CHECK (char_length(name) BETWEEN 1 AND 30),
    CONSTRAINT tags_color_hex CHECK (color ~ '^#[0-9a-fA-F]{8}$'),
    CONSTRAINT tags_created_by_name_key UNIQUE (created_by, name)
);

CREATE TABLE note_tags (
    note_id UUID NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags (id) ON DELETE CASCADE,

    PRIMARY KEY (note_id, tag_id)
);

CREATE INDEX note_tags_tag_id_idx ON note_tags (tag_id);