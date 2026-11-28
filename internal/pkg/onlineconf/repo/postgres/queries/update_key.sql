UPDATE onlineconf_keys
SET scope = $1,
    key = $2,
    description = $3,
    is_public = $4,
    value_type = $5,
    value = $6,
    enabled = $7,
    rollout_percent = $8,
    updated_by = $9,
    updated_at = now(),
    version = version + 1
WHERE id = $10 AND version = $11;
