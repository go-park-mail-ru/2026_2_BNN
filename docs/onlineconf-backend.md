# OnlineConf — ТЗ бэкенда

Конфиги и фича-флаги управляются из отдельной админки (`/onlineconf/`) и применяются в продукте без деплоя, не позже чем через 1 минуту.

## 1. Компоненты

| Компонент | Что делает |
|---|---|
| **Сервис `onlineconf`** (новый, gRPC `:9004`) | хранит ключи, ведёт историю, держит снимок всех ключей в памяти |
| **main** | HTTP API админки и runtime, проверка прав, кеш снимка и списка админов |
| **auth** | поле `users.is_admin`, поиск пользователей, список админов |
| **Пакет `internal/pkg/onlineconf`** | клиент с кешем в памяти и функция вычисления флага; используется в main и в сервисе `onlineconf` |

На пользовательских запросах в базу OnlineConf никто не ходит: всё читается из памяти.

## 2. Данные

```sql
ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE onlineconf_keys (
    id              UUID PRIMARY KEY,
    scope           TEXT NOT NULL,              -- frontend | backend | auth | notes
    key             TEXT NOT NULL,
    kind            TEXT NOT NULL,              -- config | flag
    description     TEXT NOT NULL DEFAULT '',
    is_public       BOOLEAN NOT NULL DEFAULT false, -- отдавать ли продуктовому фронту
    value_type      TEXT,                       -- config: string | number | boolean | object | array
    value           JSONB,                      -- config: значение
    enabled         BOOLEAN,                    -- flag: общий рубильник
    rollout_percent INT CHECK (rollout_percent BETWEEN 0 AND 100), -- flag
    version         INT NOT NULL DEFAULT 1,
    updated_by      UUID NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scope, key)
);

CREATE TABLE onlineconf_overrides (
    key_id   UUID REFERENCES onlineconf_keys (id) ON DELETE CASCADE,
    user_id  UUID NOT NULL,
    login    TEXT NOT NULL,
    enabled  BOOLEAN NOT NULL,                  -- true = include, false = exclude
    PRIMARY KEY (key_id, user_id)
);

CREATE TABLE onlineconf_history (
    id          UUID PRIMARY KEY,
    key_id      UUID NOT NULL,
    action      TEXT NOT NULL,                  -- create | update | delete
    old_value   JSONB,
    new_value   JSONB,
    user_id     UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE onlineconf_revision (
    id        INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    revision  BIGINT NOT NULL DEFAULT 0
);
INSERT INTO onlineconf_revision DEFAULT VALUES;
```

- Каждое изменение (создание, правка, удаление) в одной транзакции: пишет ключ, пишет историю, увеличивает `version` ключа и общую `revision`.
- Первый админ назначается вручную: `UPDATE users SET is_admin = true WHERE login = '...';`.

## 3. Вычисление флага

Одна функция в `internal/pkg/onlineconf`, её используют и продукт, и проверка пользователя в админке:

```
1. enabled = false           → false (для всех, включая include)
2. пользователь в exclude    → false
3. пользователь в include    → true
4. fnv32a(key + ":" + user_id) % 100 < rollout_percent → true, иначе false
```

Хеш детерминированный: пользователь всегда в одной и той же «корзине», при росте процента включённых только прибавляется. 0% — никому, 100% — всем, кроме exclude.

## 4. Кеш и доставка (≤ 1 минуты)

- Сервис `onlineconf` держит снимок всех ключей в памяти и перечитывает базу после каждого своего изменения и раз в 10 с.
- main раз в **15 с** вызывает `GetSnapshot(revision)`. Если ревизия не изменилась, ответ пустой. Снимок хранится в памяти (`atomic.Pointer`).
- main раз в **15 с** обновляет кеш админов: множество `user_id` с `is_admin = true` из auth.
- Продуктовый фронт опрашивает runtime раз в 15 с. Итого изменение доходит до пользователя за ≤ 30 с.
- Если `onlineconf` недоступен, используется последний полученный снимок. До первого снимка — значения по умолчанию из кода.

## 5. Доступ

- `/api/onlineconf/admin/*` — middleware в main: валидная кука **и** `user_id` в кеше админов. Нет куки → 401, не админ → 403.
- `/api/onlineconf/runtime` — любой вошедший пользователь, `user_id` только из куки.
- Сервис `onlineconf` наружу не открыт, ходит в него только main по gRPC.
- На все ответы OnlineConf: `Cache-Control: no-store`.

## 6. HTTP API

Ошибки OnlineConf возвращаются в формате `{"code": "...", "message": "..."}`.

### Админка — `/api/onlineconf/admin`

| Метод и путь | Что делает |
|---|---|
| `GET /session` | проверка доступа при открытии админки: админ — `200 {id, login}`, не админ — `403`, нет куки — `401`. Признак админа в ответе не передаётся: доступ определяет код ответа |
| `GET /scopes` | `["frontend", "backend", "auth", "notes"]` |
| `GET /keys?query=&scope=&kind=` | каталог; `query` — поиск по части имени без учёта регистра; до 200 записей, без пагинации. Ответ — массив ключей в модели ниже, но у флагов без `include_users` и `exclude_users` |
| `POST /keys` | создать ключ |
| `GET /keys/{id}` | карточка ключа |
| `PUT /keys/{id}` | сохранить ключ целиком; в теле обязательна текущая `version` |
| `DELETE /keys/{id}?version=` | удалить |
| `GET /users?query=` | поиск пользователей по логину для include/exclude: `[{id, login}]`, до 10 записей |
| `GET /keys/{id}/history` | история изменений, сначала новые: `[{id, action, old_value, new_value, user: {id, login}, created_at}]`; `action` — `create`, `update`, `delete`; `old_value` и `new_value` — ключ целиком в модели ниже (`null` при создании и удалении соответственно) *(доп.)* |
| `POST /keys/{id}/evaluate` | `{user_id}` → `{result, reason, revision}`, где `reason`: `disabled`, `excluded`, `included`, `rollout` *(доп.)* |
| `GET /admins` | список админов: `[{id, login}]` *(доп.)* |
| `PUT /admins/{user_id}` | выдать права админа; без тела, ответ `204`; `404`, если пользователя нет *(доп.)* |
| `DELETE /admins/{user_id}` | забрать права админа; без тела, ответ `204`; забрать права у самого себя нельзя — `400` *(доп.)* |

**Модель ключа** (ответ `GET /keys/{id}`; в `POST` и `PUT` те же поля, кроме `id`, `updated_*`):

```json
{
  "id": "uuid",
  "scope": "backend",
  "key": "attachments.upload_enabled",
  "kind": "flag",
  "description": "Доступность новых загрузок вложений",
  "is_public": true,
  "version": 3,
  "updated_at": "2026-11-28T09:00:00Z",
  "updated_by": {"id": "uuid", "login": "admin"},
  "flag": {
    "enabled": true,
    "rollout_percent": 25,
    "include_users": [{"id": "uuid", "login": "liza"}],
    "exclude_users": []
  }
}
```

Для `kind = config` вместо `flag`: `"config": {"value_type": "number", "value": 10485760}`. В `POST`/`PUT` вместо `include_users`/`exclude_users` передаются `include_user_ids`/`exclude_user_ids`.

**Валидация:**
- `key`: `^[a-zA-Z][a-zA-Z0-9_.-]{0,99}$`, уникален в пределах `scope`;
- `scope` — только из списка; `kind` после создания не меняется;
- `rollout_percent` — целое 0–100;
- `include` и `exclude` не пересекаются, не больше 100 пользователей в каждом, все пользователи существуют;
- `value` соответствует `value_type`; для известных ключей сервер проверяет допустимый диапазон (лимит вложений: 1–10 МиБ).

**Коды ответов:**

| Код | `code` | Когда |
|---|---|---|
| 400 | `VALIDATION` | неверные поля, JSON, диапазон |
| 401 | — | нет куки |
| 403 | `FORBIDDEN` | не админ |
| 404 | `NOT_FOUND` | ключа нет |
| 409 | `KEY_EXISTS` | такой `scope + key` уже есть |
| 409 | `VERSION_CONFLICT` | `version` устарела: ключ уже изменили в другой вкладке |

### Продукт — `GET /api/onlineconf/runtime`

Ключи скоупа `frontend` и ключи с `is_public = true`; флаги уже вычислены для текущего пользователя. Списки include/exclude и непубличные ключи не отдаются.

```json
{
  "revision": 42,
  "user_id": "uuid",
  "configs": {"backend": {"attachments.max_file_size_bytes": 10485760}},
  "flags":   {"backend": {"attachments.upload_enabled": true}}
}
```

## 7. gRPC

**`onlineconf.proto`:** `GetSnapshot`, `ListKeys`, `GetKey`, `CreateKey`, `UpdateKey`, `DeleteKey`, `ListHistory`, `Evaluate`. В изменяющие rpc main передаёт `user_id` админа — для `updated_by` и истории.

**`auth.proto`, новые rpc:** `ListAdmins`, `SetAdmin`, `SearchUsers`.

## 8. Интеграция в продукт

| Ключ | Тип, по умолчанию | Бэкенд |
|---|---|---|
| `backend` / `attachments.max_file_size_bytes` | config, number, `10485760`, public | лимит в `UploadAttachment` (main) берётся из кеша вместо константы `maxFileSize10MB`; при превышении — **413** `{"code": "FILE_TOO_LARGE", "limit_bytes": N}` |
| `backend` / `attachments.upload_enabled` | flag, `enabled = true`, 100%, public | `UploadAttachment` проверяет флаг для пользователя; если выключен — **403** `{"code": "FEATURE_DISABLED"}`. Чтение и удаление уже загруженных вложений работают как раньше |

Проверки прав на заметку и форматов файлов остаются без изменений.

## 9. Деплой

- `docker-compose`: сервис `onlineconf` (`ONLINECONF_GRPC_ADDR=onlineconf:9004` для main).
- nginx: `location /onlineconf/` отдаёт статику админки, для неизвестных путей — `try_files $uri /onlineconf/index.html`. `/api/` уже проксируется в main.
- Миграция `000009_onlineconf` (`000008` занята API-токенами).

## 10. Приоритеты

- **P0 (обязательно):** таблицы, `is_admin` и проверка прав, CRUD ключей с поиском и скоупами, вычисление флага, кеш с обновлением ≤ 1 минуты, runtime, интеграция двух ключей во вложения, деплой.
- **P1:** история изменений, проверка пользователя (`evaluate`).
- **P2:** управление админами, плавная раскатка по регламенту, автоматическая раскатка с подтверждением, плановая раскатка.
- **P3:** RBAC и группы пользователей.

## 11. Распределение работ

Принцип деления: **Лиза — ядро OnlineConf** (сервис, хранение, админское API ключей), **Сейран — доступ, доставка в продукт и деплой** (кто может войти, как значения попадают в сервисы и на фронт, где всё работает).

### Вместе, 10:00–10:30

- зафиксировать `onlineconf.proto` (сообщения ключа и снимка) и пути в OpenAPI;
- договориться о моделях `models.OnlineConfKey` и `models.Snapshot`: от них зависят обе части.

### Лиза — ядро OnlineConf

| P | Задача |
|---|---|
| P0 | миграция таблиц `onlineconf_keys`, `onlineconf_overrides`, `onlineconf_history`, `onlineconf_revision` |
| P0 | сервис `onlineconf`: `cmd/onlineconf`, repo, usecase, gRPC-обработчик и клиент |
| P0 | CRUD ключей: валидация (имя, scope, тип значения, проценты, пересечение include/exclude), `version` и `VERSION_CONFLICT`, `KEY_EXISTS`, увеличение `revision` |
| P0 | `GetSnapshot(revision)` и снимок в памяти сервиса |
| P0 | HTTP в main: `GET/POST /keys`, `GET/PUT/DELETE /keys/{id}`, `GET /scopes` (OpenAPI + обработчики) |
| P1 | запись истории в каждом изменении и `GET /keys/{id}/history` |
| P2 | регламенты, плавная и автоматическая раскатка, плановая раскатка, фоновая задача |
| P3 | группы пользователей |

### Сейран — доступ, доставка в продукт, деплой

| P | Задача |
|---|---|
| P0 | миграция `users.is_admin`, rpc в auth: `ListAdmins`, `SearchUsers` |
| P0 | кеш админов в main и middleware на `/api/onlineconf/admin/*`, `GET /session`, `GET /users` |
| P0 | пакет `internal/pkg/onlineconf`: функция вычисления флага и клиент со снимком в памяти (опрос раз в 15 с) |
| P0 | `GET /api/onlineconf/runtime` |
| P0 | интеграция во вложения: `attachments.max_file_size_bytes` и `attachments.upload_enabled`, коды `FILE_TOO_LARGE` и `FEATURE_DISABLED` |
| P0 | деплой: compose (`onlineconf`), nginx `/onlineconf/`, Prometheus, `.env`, миграции и первый админ на сервере, проверка доставки ≤ 1 минуты |
| P1 | `POST /keys/{id}/evaluate`: считается в main по тому же снимку и той же функции |
| P2 | управление админами: `SetAdmin`, `GET/PUT/DELETE /admins` |
| P3 | RBAC: права, проверка в middleware, `permissions` в `/session` |

### Зависимости

- Сейрану для клиента со снимком нужен `GetSnapshot` от Лизы. До его готовности клиент пишется и тестируется на моке gRPC.
- Лизе для `updated_by` нужен `user_id` админа из middleware Сейрана. До его готовности ручки можно проверять на любом вошедшем пользователе.
- Деплой проверяем рано (к обеду) на каркасе сервиса, а не в конце дня.

### План дня

| Время | Что готово |
|---|---|
| 10:30–13:30 | Лиза: таблицы, сервис, CRUD. Сейран: `is_admin`, middleware, пакет `onlineconf`, ранний деплой каркаса |
| 13:30–16:00 | Лиза: HTTP-ручки ключей, история. Сейран: runtime, интеграция во вложения, `evaluate` |
| 16:00–17:00 | совместная проверка с фронтом, исправления P0 |
| 17:00–18:00 | финальный деплой, данные для демо, репетиция; новых фич не начинаем |

## 12. Ручки доп. фич

Все под `/api/onlineconf/admin`.

### Регламенты, плавная и автоматическая раскатка

Регламент — это шаги процентов и минимальная пауза между ними, например `[5, 25, 50, 100]` и 30 минут.

```sql
CREATE TABLE onlineconf_rollout_plans (
    id            UUID PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE,
    steps         INT[] NOT NULL,             -- по возрастанию, последний = 100
    interval_min  INT NOT NULL                -- пауза между шагами, минуты
);

ALTER TABLE onlineconf_keys
    ADD COLUMN rollout_plan_id UUID REFERENCES onlineconf_rollout_plans (id),
    ADD COLUMN rollout_step    INT,           -- индекс текущего шага
    ADD COLUMN step_ready_at   TIMESTAMPTZ;   -- когда можно переходить дальше
```

| Метод и путь | Что делает |
|---|---|
| `GET /rollout-plans` | список регламентов |
| `POST /rollout-plans` | создать `{name, steps, interval_min}` |
| `PUT /rollout-plans/{id}`, `DELETE /rollout-plans/{id}` | изменить, удалить (если не привязан к флагам) |
| `PUT /keys/{id}/rollout-plan` | `{plan_id}` — привязать регламент к флагу, раскатка начинается с первого шага; `null` — отвязать |
| `GET /keys/{id}/rollout` | состояние: текущий шаг, следующий шаг, `step_ready_at`, `status`: `waiting`, `pending_approval`, `done` |
| `POST /keys/{id}/rollout/approve` | перейти на следующий шаг; только при `pending_approval` |
| `POST /keys/{id}/rollout/rollback` | вернуться на предыдущий шаг |

- **Плавная раскатка:** если к флагу привязан регламент, `PUT /keys/{id}` с `rollout_percent`, которого нет в шагах или который перескакивает шаг, отклоняется с `400 ROLLOUT_STEP_NOT_ALLOWED`.
- **Автоматическая с подтверждением:** фоновая задача в сервисе `onlineconf` раз в минуту переводит флаги, у которых наступил `step_ready_at`, в `pending_approval`. Процент меняется только после `approve`.

### Плановая раскатка

```sql
CREATE TABLE onlineconf_schedules (
    id              UUID PRIMARY KEY,
    key_id          UUID REFERENCES onlineconf_keys (id) ON DELETE CASCADE,
    run_at          TIMESTAMPTZ NOT NULL,
    enabled         BOOLEAN,
    rollout_percent INT,
    done            BOOLEAN NOT NULL DEFAULT false,
    created_by      UUID NOT NULL
);
```

| Метод и путь | Что делает |
|---|---|
| `GET /keys/{id}/schedules` | запланированные изменения флага |
| `POST /keys/{id}/schedules` | `{run_at, enabled?, rollout_percent?}`; `run_at` в будущем, в UTC |
| `DELETE /keys/{id}/schedules/{schedule_id}` | отменить, пока не выполнено |

Выполняет фоновая задача сервиса `onlineconf` (раз в минуту), изменение пишется в историю от имени создателя.

### Группы пользователей

```sql
CREATE TABLE onlineconf_groups (id UUID PRIMARY KEY, name TEXT NOT NULL UNIQUE);
CREATE TABLE onlineconf_group_members (
    group_id UUID REFERENCES onlineconf_groups (id) ON DELETE CASCADE,
    user_id  UUID NOT NULL,
    PRIMARY KEY (group_id, user_id)
);
```

| Метод и путь | Что делает |
|---|---|
| `GET /groups`, `POST /groups` | список, создать `{name}` |
| `PUT /groups/{id}`, `DELETE /groups/{id}` | переименовать, удалить |
| `GET /groups/{id}/members` | участники `[{id, login}]` |
| `PUT /groups/{id}/members/{user_id}`, `DELETE /groups/{id}/members/{user_id}` | добавить, убрать участника |

### RBAC

Право — это субъект (пользователь или группа), область действия (весь scope или один ключ) и набор действий.

```sql
CREATE TABLE onlineconf_permissions (
    id            UUID PRIMARY KEY,
    subject_type  TEXT NOT NULL,              -- user | group
    subject_id    UUID NOT NULL,
    scope         TEXT NOT NULL,
    key_id        UUID REFERENCES onlineconf_keys (id) ON DELETE CASCADE, -- NULL = весь scope
    actions       TEXT[] NOT NULL             -- create | read | update | delete
);
```

| Метод и путь | Что делает |
|---|---|
| `GET /permissions?subject_type=&subject_id=` | список прав |
| `POST /permissions` | `{subject_type, subject_id, scope, key_id?, actions}` |
| `DELETE /permissions/{id}` | отозвать право |

- Итоговые права пользователя = его права + права всех его групп. `is_admin` даёт всё.
- Каждая ручка `/keys/*` проверяет нужное действие на сервере; при отказе — `403 FORBIDDEN`.
- `GET /session` пускает пользователя, у которого есть хотя бы одно право, и отдаёт `permissions: [{scope, key_id, actions}]`, чтобы фронт скрывал недоступные кнопки. Это только подсказка интерфейсу: каждое действие всё равно проверяется на сервере.
- Список прав кешируется в main вместе со списком админов (15 с).
