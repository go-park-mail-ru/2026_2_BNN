# 3 Нормальная форма и НФБК

Блок может использоваться в нескольких заметках; вложение — в нескольких блоках.
Внутри одного родителя элемент встречается один раз; position хранит его порядок.
content — единое значение редактора, без самостоятельных сущностей/связей внутри.
login уникален и обязателен; иных функциональных зависимостей не предполагается.

Для 3НФ из `note` убраны `creator_login` и `creator_avatar`. Они зависят от пользователя, а не от самой заметки, и уже есть в `user`.

P.S.
Почему у нас получились 3НФ и НФБК в одной схеме. НФБК строже 3НФ: левая часть каждой зависимости должна быть ключом. Разница появляется, только если часть ключа зависит от атрибута, который сам ключом не является. После выноса данных автора таких зависимостей нет: всё определяется ключами таблиц. Поэтому отдельно дробить схему для НФБК не является нужным.

История блока — `block_version`. Снимок определяет пара `{block_id, version}`, это ключ, поэтому отношение сразу на итоговой схеме.

`_` перед именем атрибута — не тип данных. Без этого слова mermaid не рисует атрибут: строка `id PK` для него синтаксическая ошибка.

```mermaid
erDiagram
    user ||..o{ note : "создаёт"
    note |o..o{ note : "родитель"
    user ||--o{ favorite : "избирает"
    note ||--o{ favorite : "избрана"
    note ||--o{ note_block : "содержит"
    block ||--o{ note_block : "входит в"
    block ||--o{ block_attachment : "имеет"
    block ||--o{ block_version : "имеет версии"
    user ||..o{ block_version : "записывает"
    attachment ||--o{ block_attachment : "прикреплено"

    user {
        _ id PK
        _ login UK
        _ password_hash
        _ avatar
        _ created_at
        _ updated_at
    }

    note {
        _ id PK
        _ parent_note_id FK
        _ created_by FK
        _ title
        _ header
        _ icon
        _ created_at
        _ updated_at
    }

    favorite {
        _ user_id PK, FK
        _ note_id PK, FK
    }

    block {
        _ id PK
        _ content
        _ created_at
        _ updated_at
    }

    block_version {
        _ block_id PK, FK
        _ version PK "номер версии вместе с block_id"
        _ content
        _ created_by FK
        _ created_at
    }

    note_block {
        _ note_id PK, FK
        _ block_id PK, FK
        _ position "уникален вместе с note_id"
    }

    attachment {
        _ id PK
        _ path
    }

    block_attachment {
        _ block_id PK, FK
        _ attachment_id PK, FK
        _ position "уникален вместе с block_id"
    }
```
