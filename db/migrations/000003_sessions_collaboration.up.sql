-- Сессии, посещения и совместная работа.
-- Правила выдачи доступа и отправки сообщений описаны в relations.md.
-- Время хранится в TIMESTAMPTZ;

CREATE TABLE user_session (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user" (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    token_hash TEXT NOT NULL UNIQUE CHECK (char_length(btrim(token_hash)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMPTZ,
    CHECK (expires_at > created_at),
    CHECK (last_seen_at >= created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX user_session_user_idx ON user_session (user_id);

CREATE TABLE user_visit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES user_session (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMPTZ,
    CHECK (last_activity_at >= started_at),
    CHECK (ended_at IS NULL OR ended_at = last_activity_at)
);
CREATE UNIQUE INDEX user_visit_one_open_per_session
    ON user_visit (session_id) WHERE ended_at IS NULL;

CREATE TABLE note_member (
    note_id UUID NOT NULL REFERENCES note (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    user_id UUID NOT NULL REFERENCES "user" (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    role TEXT NOT NULL CHECK (role IN ('editor', 'viewer')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    removed_at TIMESTAMPTZ,
    PRIMARY KEY (note_id, user_id),
    CHECK (removed_at IS NULL OR removed_at >= joined_at)
);
CREATE INDEX note_member_user_idx ON note_member (user_id);

CREATE TABLE note_invitation (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID NOT NULL REFERENCES note (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    invited_by UUID NOT NULL REFERENCES "user" (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    invited_user_id UUID NOT NULL REFERENCES "user" (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    role TEXT NOT NULL CHECK (role IN ('editor', 'viewer')),
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'declined', 'revoked', 'expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    responded_at TIMESTAMPTZ,
    CHECK (invited_by <> invited_user_id),
    CHECK (expires_at > created_at),
    CHECK (responded_at IS NULL OR responded_at >= created_at),
    CHECK (
        (status IN ('accepted', 'declined') AND responded_at IS NOT NULL)
        OR (status IN ('pending', 'revoked', 'expired') AND responded_at IS NULL)
    )
);
CREATE UNIQUE INDEX note_invitation_one_pending
    ON note_invitation (note_id, invited_user_id) WHERE status = 'pending';

CREATE TABLE chat (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id UUID NOT NULL UNIQUE REFERENCES note (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chat_message (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_id UUID NOT NULL REFERENCES chat (id) ON DELETE CASCADE ON UPDATE RESTRICT,
    sender_id UUID NOT NULL REFERENCES "user" (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    content TEXT NOT NULL CHECK (char_length(btrim(content)) > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (updated_at >= created_at)
);
CREATE INDEX chat_message_chat_time_idx ON chat_message (chat_id, created_at, id);
CREATE TRIGGER chat_message_set_updated_at
    BEFORE UPDATE ON chat_message
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Начальные связи для заметок, созданных до этой миграции.
INSERT INTO note_member (note_id, user_id, role)
SELECT id, created_by, 'editor' FROM note;

INSERT INTO chat (note_id)
SELECT id FROM note;
