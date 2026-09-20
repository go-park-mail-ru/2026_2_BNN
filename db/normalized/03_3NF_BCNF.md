# 3 Нормальная форма и НФБК

Блок может использоваться в нескольких заметках; вложение — в нескольких блоках.
Внутри одного родителя элемент встречается один раз; position хранит его порядок.
content — единое значение редактора, без самостоятельных сущностей/связей внутри.
login уникален и обязателен; иных функциональных зависимостей не предполагается.

Для 3НФ из `note` убраны `creator_login` и `creator_avatar`. Они зависят от пользователя, а не от самой заметки, и уже есть в `user`.

P.S.
Почему у нас получились 3НФ и НФБК в одной схеме. НФБК строже 3НФ: левая часть каждой зависимости должна быть ключом. Разница появляется, только если часть ключа зависит от атрибута, который сам ключом не является. После выноса данных автора таких зависимостей нет: всё определяется ключами таблиц. Поэтому отдельно дробить схему для НФБК не является нужным.

```mermaid
erDiagram
    user ||..o{ note : "создаёт"
    note |o..o{ note : "родитель"
    user ||--o{ favorite : "избирает"
    note ||--o{ favorite : "избрана"
    note ||--o{ note_block : "содержит"
    block ||--o{ note_block : "входит в"
    block ||--o{ block_attachment : "имеет"
    attachment ||--o{ block_attachment : "прикреплено"

    user {
        uuid id PK
        text login UK
        text password_hash
        text avatar
        timestamptz created_at
        timestamptz updated_at
    }

    note {
        uuid id PK
        uuid parent_note_id FK
        uuid created_by FK
        text title
        text header
        text icon
        timestamptz created_at
        timestamptz updated_at
    }

    favorite {
        uuid user_id PK, FK
        uuid note_id PK, FK
    }

    block {
        uuid id PK
        text content
        timestamptz created_at
        timestamptz updated_at
    }

    note_block {
        uuid note_id PK, FK
        uuid block_id PK, FK
        integer position UK "уникален вместе с note_id"
    }

    attachment {
        uuid id PK
        text path
    }

    block_attachment {
        uuid block_id PK, FK
        uuid attachment_id PK, FK
        integer position UK "уникален вместе с block_id"
    }
```
