UPDATE onlineconf_keys SET rollout_pending_approval = true
WHERE rollout_plan_id IS NOT NULL AND NOT rollout_pending_approval
  AND step_ready_at <= clock_timestamp();
