# 2026_2_BNN
Backend проекта Notion команды BNN 🪲

## Структура проекта

```text
build/             — файлы сборки и скрипты БД
cmd/main/          — точка входа и запуск сервера
internal/models/   — структуры пользователя и записи
internal/pkg/auth/ — обработчики регистрации и входа
internal/pkg/note/ — обработчики получения записей
postman/           — коллекция запросов
```

## Схема базы данных

``` mermaid
erDiagram
    direction LR

    users {
        uuid id PK
        text login PK
        text password_hash "NOT NULL"
        text avatar "DEFAULT '/default_avatar'"
        timestamptz created_at "NOT NULL, DEFAULT now()"
        timestamptz updated_at "NOT NULL, DEFAULT now()"
    }

    notes {
        uuid id PK
        uuid parent_note_id FK
        uuid[] blocks_id
        uuid created_by FK "NOT NULL"
        varchar(255) title "NOT NULL, DEFAULT 'Без названия'"
        text header
        text icon
        timestamptz created_at "NOT NULL, DEFAULT now()"
        timestamptz updated_at "NOT NULL, DEFAULT now()"
    }

    favorite {
        uuid user_id PK, FK "NOT NULL"
        uuid note_id PK, FK "NOT NULL"
    }

    blocks {
        uuid id PK
        uuid[] attachments_id
        jsonb content "NOT NULL"
        timestamptz created_at "NOT NULL, DEFAULT now()"
        timestamptz updated_at "NOT NULL, DEFAULT now()"
    }

    attachments {
        uuid id PK
        text path
    }

    users ||--o{ notes : ""
    notes o|--o{ notes : ""
    users ||--o{ favorite : ""
    notes ||--o{ favorite : ""

```




