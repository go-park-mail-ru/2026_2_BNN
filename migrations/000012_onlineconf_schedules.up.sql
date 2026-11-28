ALTER TABLE onlineconf_keys
    ADD COLUMN rollout_pending_approval BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX onlineconf_keys_rollout_ready_idx
    ON onlineconf_keys (step_ready_at)
    WHERE rollout_plan_id IS NOT NULL AND NOT rollout_pending_approval;

CREATE TABLE onlineconf_schedules (
    id UUID PRIMARY KEY,
    key_id UUID NOT NULL REFERENCES onlineconf_keys (id) ON DELETE CASCADE,
    run_at TIMESTAMPTZ NOT NULL,
    enabled BOOLEAN,
    rollout_percent INT CHECK (rollout_percent BETWEEN 0 AND 100),
    done BOOLEAN NOT NULL DEFAULT false,
    failed BOOLEAN NOT NULL DEFAULT false,
    error TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (enabled IS NOT NULL OR rollout_percent IS NOT NULL),
    CHECK (NOT (done AND failed))
);

CREATE INDEX onlineconf_schedules_due_idx
    ON onlineconf_schedules (run_at, id) WHERE NOT done AND NOT failed;
CREATE INDEX onlineconf_schedules_key_idx
    ON onlineconf_schedules (key_id, run_at, id);
