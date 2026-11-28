CREATE TABLE onlineconf_rollout_plans (
    id           UUID PRIMARY KEY,
    name         TEXT NOT NULL,
    steps        INT[] NOT NULL,
    interval_min INT NOT NULL,

    CONSTRAINT onlineconf_rollout_plans_name_key
        UNIQUE (name),

    CONSTRAINT onlineconf_rollout_plans_name_check
        CHECK (
            name = btrim(name)
            AND char_length(name) BETWEEN 1 AND 100
        ),

    CONSTRAINT onlineconf_rollout_plans_steps_check
        CHECK (
            array_ndims(steps) = 1
            AND array_lower(steps, 1) = 1
            AND cardinality(steps) BETWEEN 2 AND 101
            AND array_position(steps, NULL) IS NULL
            AND 0 <= ALL (steps)
            AND 100 >= ALL (steps)
            AND steps[cardinality(steps)] = 100
        ),

    CONSTRAINT onlineconf_rollout_plans_interval_check
        CHECK (interval_min > 0)
);