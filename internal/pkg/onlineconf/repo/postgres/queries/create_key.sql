INSERT INTO onlineconf_keys (
    id, scope, key, kind, description, is_public,
    value_type, value, enabled, rollout_percent, updated_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
