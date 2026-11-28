SELECT version
FROM onlineconf_keys
WHERE id = $1
FOR UPDATE;
