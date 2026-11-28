INSERT INTO onlineconf_schedules (id, key_id, run_at, enabled, rollout_percent, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, key_id, run_at, enabled, rollout_percent, done, failed, error, created_by, created_at;
