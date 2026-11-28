INSERT INTO onlineconf_rollout_plans (id, name, steps, interval_min)
VALUES ($1, $2, $3, $4)
RETURNING id, name, steps, interval_min;
