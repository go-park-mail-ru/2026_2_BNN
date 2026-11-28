SELECT COALESCE(rollout_pending_approval AND step_ready_at <= clock_timestamp(), false)
FROM onlineconf_keys WHERE id = $1;
