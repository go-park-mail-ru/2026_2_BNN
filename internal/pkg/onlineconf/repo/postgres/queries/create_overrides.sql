INSERT INTO onlineconf_overrides (key_id, user_id, login, enabled)
SELECT $1, o.user_id, o.login, o.enabled
FROM unnest($2::uuid[], $3::text[], $4::boolean[]) AS o(user_id, login, enabled);
