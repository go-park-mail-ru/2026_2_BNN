DROP TABLE onlineconf_schedules;
DROP INDEX onlineconf_keys_rollout_ready_idx;
ALTER TABLE onlineconf_keys DROP COLUMN rollout_pending_approval;
