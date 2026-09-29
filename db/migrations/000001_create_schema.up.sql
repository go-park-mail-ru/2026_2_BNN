-- Нет DEFAULT у login, password_hash, avatar, title, header, icon, content,
-- path, position, version и внешних ключей: значение задаёт приложение.
-- avatar, title, header и icon могут быть NULL.
-- parent_note_id может быть NULL, остальные внешние ключи — нет.
--
-- Удаление автора не удаляет заметки (created_by ON DELETE RESTRICT)
-- и не трогает блоки и вложения: они автору не принадлежат.
-- Избранное автора удаляется (ON DELETE CASCADE).
-- Удаление родителя обнуляет parent_note_id у дочерних заметок (ON DELETE SET NULL).
-- Удаление заметки снимает её избранное и note_block, блоки остаются.
-- Удаление блока снимает note_block, block_attachment и block_version, attachment остаётся.
-- Удаление автора версии запрещено, пока строки block_version на него ссылаются
-- (created_by ON DELETE RESTRICT).
-- Удаление attachment снимает только block_attachment.
-- Идентификаторы не переписываются: ON UPDATE RESTRICT.
--
-- У favorite, note_block и block_attachment нет created_at и updated_at:
-- это связи, своей истории времени у строки нет.
-- У block_version нет updated_at: снимок после записи не меняется.
--
-- DEFAULT выставляет updated_at только при INSERT.
-- При UPDATE его обновляет триггер: ограничением это не выразить.

CREATE FUNCTION set_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;

CREATE TABLE user (
    PRIMARY KEY (id),
    id            UUID        DEFAULT gen_random_uuid() NOT NULL,
    login         TEXT                                  NOT NULL UNIQUE,
                  CONSTRAINT user_login_not_empty
                  CHECK (char_length(btrim(login)) > 0),
    password_hash TEXT                                  NOT NULL,
    avatar        TEXT,
    created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TRIGGER user_set_updated_at
    BEFORE UPDATE ON user
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE note (
    PRIMARY KEY (id),
    id             UUID        DEFAULT gen_random_uuid() NOT NULL,
    parent_note_id UUID,
                   CONSTRAINT note_parent_not_self
                   CHECK (parent_note_id IS DISTINCT FROM id),
    created_by     UUID                                  NOT NULL,
    title          TEXT,
    header         TEXT,
    icon           TEXT,
    created_at     TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at     TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (parent_note_id) REFERENCES note (id)
        ON DELETE SET NULL
        ON UPDATE RESTRICT,
    FOREIGN KEY (created_by) REFERENCES user (id)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT
);

CREATE TRIGGER note_set_updated_at
    BEFORE UPDATE ON note
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE favorite (
    PRIMARY KEY (user_id, note_id),
    user_id UUID NOT NULL,
    note_id UUID NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,
    FOREIGN KEY (note_id) REFERENCES note (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT
);

CREATE TABLE block (
    PRIMARY KEY (id),
    id         UUID        DEFAULT gen_random_uuid() NOT NULL,
    content    TEXT                                  NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TRIGGER block_set_updated_at
    BEFORE UPDATE ON block
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE block_version (
    PRIMARY KEY (block_id, version),
    block_id   UUID                                  NOT NULL,
    version    INTEGER                               NOT NULL,
    content    TEXT                                  NOT NULL,
    created_by UUID                                  NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    FOREIGN KEY (block_id) REFERENCES block (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,
    FOREIGN KEY (created_by) REFERENCES user (id)
        ON DELETE RESTRICT
        ON UPDATE RESTRICT,
    CONSTRAINT block_version_number_positive
        CHECK (version >= 1)
);

CREATE TABLE note_block (
    PRIMARY KEY (note_id, block_id),
    note_id  UUID    NOT NULL,
    block_id UUID    NOT NULL,
    position INTEGER NOT NULL,
    UNIQUE (note_id, position),
    FOREIGN KEY (note_id) REFERENCES note (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,
    FOREIGN KEY (block_id) REFERENCES block (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,
    CONSTRAINT note_block_position_non_negative
        CHECK (position >= 0)
);

CREATE TABLE attachment (
    PRIMARY KEY (id),
    id         UUID        DEFAULT gen_random_uuid() NOT NULL,
    path       TEXT                                  NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TRIGGER attachment_set_updated_at
    BEFORE UPDATE ON attachment
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TABLE block_attachment (
    PRIMARY KEY (block_id, attachment_id),
    block_id      UUID    NOT NULL,
    attachment_id UUID    NOT NULL,
    position      INTEGER NOT NULL,
    UNIQUE (block_id, position),
    FOREIGN KEY (block_id) REFERENCES block (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,
    FOREIGN KEY (attachment_id) REFERENCES attachment (id)
        ON DELETE CASCADE
        ON UPDATE RESTRICT,
    CONSTRAINT block_attachment_position_non_negative
        CHECK (position >= 0)
);
