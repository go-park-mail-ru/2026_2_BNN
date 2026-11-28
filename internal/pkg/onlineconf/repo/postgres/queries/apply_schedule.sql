UPDATE onlineconf_keys
SET enabled = COALESCE($2, enabled), rollout_percent = COALESCE($3, rollout_percent),
    updated_by = $4, updated_at = clock_timestamp(), version = version + 1
WHERE id = $1;
