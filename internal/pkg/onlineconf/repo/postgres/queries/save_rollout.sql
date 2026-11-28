UPDATE onlineconf_keys
SET rollout_plan_id = $2, rollout_step = $3, rollout_percent = $4,
    step_ready_at = CASE WHEN $5::boolean THEN clock_timestamp() + make_interval(mins => $6::integer) ELSE NULL END,
    rollout_pending_approval = false, updated_by = $7,
    updated_at = clock_timestamp(), version = version + 1
WHERE id = $1;
