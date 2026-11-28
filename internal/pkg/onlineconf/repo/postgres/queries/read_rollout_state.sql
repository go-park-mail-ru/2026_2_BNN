SELECT k.id, k.kind, COALESCE(k.rollout_percent, 0), k.version,
       k.rollout_plan_id, k.rollout_step, k.step_ready_at, k.rollout_pending_approval,
       p.name, p.steps, p.interval_min
FROM onlineconf_keys k
LEFT JOIN onlineconf_rollout_plans p ON p.id = k.rollout_plan_id
WHERE k.id = $1;
