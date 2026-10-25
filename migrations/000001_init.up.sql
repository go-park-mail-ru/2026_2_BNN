CREATE TABLE users (
    id            UUID PRIMARY KEY,
    login         TEXT NOT NULL UNIQUE,
    password_hash BYTEA NOT NULL,
    avatar        TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_login_length CHECK (char_length(login) BETWEEN 3 AND 20)
);

CREATE TABLE refresh_tokens (
    token_hash BYTEA PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

CREATE TABLE notes (
    id             UUID PRIMARY KEY,
    parent_note_id UUID REFERENCES notes (id) ON DELETE CASCADE,
    created_by     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title          VARCHAR(255) NOT NULL DEFAULT '',
    header         TEXT NOT NULL DEFAULT '',
    icon           TEXT NOT NULL DEFAULT '',
    block_ids      UUID[] NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX notes_created_by_idx ON notes (created_by);
CREATE INDEX notes_parent_note_id_idx ON notes (parent_note_id);

CREATE TABLE blocks (
    id         UUID PRIMARY KEY,
    note_id    UUID NOT NULL REFERENCES notes (id) ON DELETE CASCADE,
    content    JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX blocks_note_id_idx ON blocks (note_id);

CREATE TABLE attachments (
    id       UUID PRIMARY KEY,
    block_id UUID NOT NULL REFERENCES blocks (id) ON DELETE CASCADE,
    path     TEXT NOT NULL UNIQUE
);

CREATE INDEX attachments_block_id_idx ON attachments (block_id);

CREATE TABLE favorite (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    note_id UUID NOT NULL REFERENCES notes (id) ON DELETE CASCADE,

    PRIMARY KEY (user_id, note_id)
);

CREATE INDEX favorite_note_id_idx ON favorite (note_id);