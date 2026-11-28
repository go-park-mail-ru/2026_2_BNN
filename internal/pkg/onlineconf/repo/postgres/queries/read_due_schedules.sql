SELECT id, key_id FROM onlineconf_schedules
WHERE NOT done AND NOT failed AND run_at <= clock_timestamp()
ORDER BY run_at, id LIMIT $1;
