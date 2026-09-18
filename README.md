# 2026_2_BNN
Backend проекта Notion команды BNN 🪲

## Структура проекта 

```
├───build (тут лежат докерфайлы)
│       main.Dockerfile (докерфайл для основного сервиса)
│       
├───cmd (тут лежат main файлы для микросервисов)
│   └───main (сервис)
│           main.go (точка входа )
│           
├───internal (внутренние файлы)
│   ├───models (модели)
│   │       user.go (модели пользователя)
│   │       note.go (модели заметок)
│   └───pkg (отдельные пакеты для логических сущностей)
│       ├───auth (пакет авторизации)
│       │       handlers.go ()
│       └───smth (пакет работы с данными, специфичными для проекта)
│               handlers.go
└───postman (здесь лежит json коллекция)
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
