UPDATE onlineconf_schedules SET failed = true, error = $2 WHERE id = $1;
