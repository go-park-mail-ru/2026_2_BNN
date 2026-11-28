SELECT EXISTS (
    SELECT 1
    FROM onlineconf_keys
    WHERE rollout_plan_id = $1
);
