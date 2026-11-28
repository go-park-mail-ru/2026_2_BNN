ALTER TABLE onlineconf_keys
    ADD COLUMN rollout_plan_id UUID,
    ADD COLUMN rollout_step INT,
    ADD COLUMN step_ready_at TIMESTAMPTZ,

    ADD CONSTRAINT onlineconf_keys_rollout_plan_id_fkey
        FOREIGN KEY (rollout_plan_id)
        REFERENCES onlineconf_rollout_plans (id)
        ON DELETE RESTRICT,

    ADD CONSTRAINT onlineconf_keys_rollout_state_check
        CHECK (
            (
                rollout_plan_id IS NULL
                AND rollout_step IS NULL
                AND step_ready_at IS NULL
            )
            OR
            (
                rollout_plan_id IS NOT NULL
                AND kind = 'flag'
                AND rollout_step IS NOT NULL
                AND rollout_step >= 0
            )
        );

CREATE INDEX onlineconf_keys_rollout_plan_id_idx
    ON onlineconf_keys (rollout_plan_id)
    WHERE rollout_plan_id IS NOT NULL;
    