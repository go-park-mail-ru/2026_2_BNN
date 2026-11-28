SELECT id
FROM onlineconf_rollout_plans
WHERE id = $1
FOR UPDATE;
