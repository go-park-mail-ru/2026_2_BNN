SELECT k.id, k.scope, k.key, k.kind, k.description, k.is_public,
       k.value_type, k.value, k.enabled, k.rollout_percent, k.version,
       k.updated_by, COALESCE(u.login, ''), k.updated_at,
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object('id', o.user_id, 'login', o.login) ORDER BY o.user_id)
           FROM onlineconf_overrides o
           WHERE o.key_id = k.id AND o.enabled
       ), '[]'::jsonb),
       COALESCE((
           SELECT jsonb_agg(jsonb_build_object('id', o.user_id, 'login', o.login) ORDER BY o.user_id)
           FROM onlineconf_overrides o
           WHERE o.key_id = k.id AND NOT o.enabled
       ), '[]'::jsonb)
FROM onlineconf_keys k
LEFT JOIN users u ON u.id = k.updated_by
ORDER BY k.scope, k.key, k.id;
