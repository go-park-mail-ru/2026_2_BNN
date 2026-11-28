CREATE TABLE onlineconf_keys (
    id              UUID PRIMARY KEY,
    scope           TEXT NOT NULL,
    key             TEXT NOT NULL,
    kind            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    is_public       BOOLEAN NOT NULL DEFAULT false,
    value_type      TEXT,
    value           JSONB,
    enabled         BOOLEAN,
    rollout_percent INT CHECK (rollout_percent BETWEEN 0 AND 100),
    version         INT NOT NULL DEFAULT 1,
    updated_by      UUID NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (scope, key)
);

CREATE TABLE onlineconf_overrides (
    key_id  UUID NOT NULL REFERENCES onlineconf_keys (id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    login   TEXT NOT NULL,
    enabled BOOLEAN NOT NULL,

    PRIMARY KEY (key_id, user_id)
);

CREATE TABLE onlineconf_history (
    id         UUID PRIMARY KEY,
    key_id     UUID NOT NULL,
    action     TEXT NOT NULL,
    old_value  JSONB,
    new_value  JSONB,
    user_id    UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE onlineconf_revision (
    id       INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    revision BIGINT NOT NULL DEFAULT 0
);

INSERT INTO onlineconf_revision DEFAULT VALUES;
