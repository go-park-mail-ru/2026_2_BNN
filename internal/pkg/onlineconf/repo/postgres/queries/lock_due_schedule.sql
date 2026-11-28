SELECT id, key_id, run_at, enabled, rollout_percent, done, failed, error, created_by, created_at
FROM onlineconf_schedules
WHERE key_id = $1 AND id = $2 AND NOT done AND NOT failed AND run_at <= clock_timestamp()
FOR UPDATE SKIP LOCKED;
