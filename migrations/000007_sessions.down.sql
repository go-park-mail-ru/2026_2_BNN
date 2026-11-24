DROP INDEX IF EXISTS refresh_tokens_id_idx;

ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS revoked_at;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS id;
