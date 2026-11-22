ALTER TABLE refresh_tokens ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE refresh_tokens ADD COLUMN revoked_at TIMESTAMPTZ;

CREATE UNIQUE INDEX refresh_tokens_id_idx ON refresh_tokens (id);