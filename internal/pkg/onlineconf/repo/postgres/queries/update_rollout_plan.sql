UPDATE onlineconf_rollout_plans
SET name = $2,
    steps = $3,
    interval_min = $4
WHERE id = $1
RETURNING id, name, steps, interval_min;
