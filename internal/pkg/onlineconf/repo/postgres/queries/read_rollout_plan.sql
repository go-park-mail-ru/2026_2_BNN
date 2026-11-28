SELECT id, name, steps, interval_min
FROM onlineconf_rollout_plans
WHERE id = $1;
