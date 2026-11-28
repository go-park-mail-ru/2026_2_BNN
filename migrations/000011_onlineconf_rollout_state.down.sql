DROP INDEX onlineconf_keys_rollout_plan_id_idx;

ALTER TABLE onlineconf_keys
    DROP CONSTRAINT onlineconf_keys_rollout_state_check,
    DROP CONSTRAINT onlineconf_keys_rollout_plan_id_fkey,
    DROP COLUMN step_ready_at,
    DROP COLUMN rollout_step,
    DROP COLUMN rollout_plan_id;