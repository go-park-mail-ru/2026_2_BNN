CREATE FUNCTION create_welcome_note() RETURNS TRIGGER AS $$
DECLARE
    welcome_note_id UUID := gen_random_uuid();
    welcome_block_id UUID;
    block_content JSONB;
BEGIN
    INSERT INTO notes (id, created_by, title, icon)
    VALUES (welcome_note_id, NEW.id, 'Добро пожаловать', '👋');

    FOR block_content IN SELECT value FROM jsonb_array_elements(jsonb_build_array(
        jsonb_build_object('type', 'heading', 'text', format('Привет, %s!', NEW.login)),
        jsonb_build_object('type', 'text', 'text', 'Это Brand New Notes — место для твоих заметок и идей. Вот что здесь можно делать:'),
        jsonb_build_object('type', 'bulleted', 'text', 'Создавать заметки и вкладывать их друг в друга'),
        jsonb_build_object('type', 'bulleted', 'text', 'Добавлять заголовки, списки, чек-листы и цитаты'),
        jsonb_build_object('type', 'bulleted', 'text', 'Прикреплять файлы и картинки прямо в текст'),
        jsonb_build_object('type', 'bulleted', 'text', 'Отмечать заметки тегами и добавлять в избранное'),
        jsonb_build_object('type', 'bulleted', 'text', 'Ставить аватарку и включать двухфакторную защиту в профиле'),
        jsonb_build_object('type', 'checklist', 'text', 'Прочитать эту заметку', 'checked', true),
        jsonb_build_object('type', 'checklist', 'text', 'Создать свою первую заметку', 'checked', false)
    ))
    LOOP
        welcome_block_id := gen_random_uuid();

        INSERT INTO blocks (id, note_id, content)
        VALUES (welcome_block_id, welcome_note_id, block_content);

        UPDATE notes SET block_ids = array_append(block_ids, welcome_block_id)
        WHERE id = welcome_note_id;
    END LOOP;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_welcome_note
AFTER INSERT ON users
FOR EACH ROW EXECUTE FUNCTION create_welcome_note();
