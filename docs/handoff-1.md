# Handoff 1 — OnlineConf, часть Сейрана (доступ к админке)

## Как работаем

- Код в проект вставляет **Сейран сам**. Ассистент пишет шаги в чате: что, зачем, код. В проекте ничего не меняет без прямой просьбы.
- Каждый шаг сначала проверяется на копии проекта в scratchpad (сборка, `go vet`, тесты, при необходимости сквозной тест через `httptest`), и только потом выдаётся.
- Защита для команд в scratchpad: `cd "$S/..." || exit 1; [ "$(pwd)" = "$S/..." ] || exit 1`.
- Правила ментора: никаких констант в `models`, никаких дублей помощников между пакетами, логика не в `main`, `samber/lo`, моки через gomock, помощники в `internal/pkg/utils/`.
- Никаких следов ассистента в коде, коммитах и PR.

## Контекст

- Бэкенд Brand New Notes, Go, один `go.mod` (`bnn`). Сервисы: **main** (HTTP `:5458`, метрики `:9101`), **auth** (gRPC `:9001`, метрики `:9102`), **notes** (gRPC `:9002`, метрики `:9103`; notes + tags + attachments). Общая база Postgres.
- Слои: `delivery/http` (oapi-codegen strict server) / `delivery/grpc` (`handler.go`, `convert/`, `gen/`) → `usecase` → `repo`. gRPC-клиенты в `internal/pkg/<домен>/client` реализуют интерфейс usecase.
- Хакатон: **OnlineConf** (конфиги и фича-флаги из отдельной админки, применение ≤ 1 минуты). ТЗ бэкенда: **`docs/onlineconf-backend.md`** (копия: `~/Desktop/Расписание/onlineconf-backend.md`). ТЗ фронта: `~/Desktop/BNN_OnlineConf_frontend_specification.pdf`.
- Делёж (раздел 11 ТЗ): **Лиза** — сервис `onlineconf` (таблицы, CRUD ключей, `GetSnapshot`, ручки `/keys`, история). **Сейран** — доступ, доставка в продукт, деплой.

## Сделано в этой сессии

Ветка **`onlyconf`** (от `main`), коммит `47a74df Add OnlineConf admin access: is_admin, admin cache, session and user search`. Сборка, `go vet` и тесты проходят.

**Шаг 1 — auth:**
- миграция `migrations/000008_users_is_admin` (`users.is_admin BOOLEAN NOT NULL DEFAULT false`);
- запросы `auth/repo/postgres/queries/read_admins.sql`, `search_users.sql` (поиск `strpos(lower(login), lower($1)) > 0`, лимит 10);
- `auth.Repo`: `ReadAdmins`, `SearchUsers(query, limit)` + общий `readShortUsers` (только `id, login`);
- `auth.Usecase`: `ListAdmins`, `SearchUsers(query)` в `auth/usecase/admins.go` (запрос 1–20 символов, иначе `ErrValidation`);
- proto: `rpc ListAdmins(Empty) returns (UsersResponse)`, `rpc SearchUsers(SearchUsersRequest) returns (UsersResponse)`; `convert.UsersFromProto`; обработчик и клиент.

**Шаг 2 — main:**
- `internal/pkg/onlineconf/admins.go`: `AdminCache` (опрос `ListAdmins` раз в 15 с, `atomic.Pointer[map[uuid.UUID]models.User]`, при ошибке остаётся старый список), интерфейс `AdminSource`;
- `internal/pkg/onlineconf/delivery/http/`: `middleware.go` (`AdminOnly`: `Cache-Control: no-store`, 401 без пользователя, 403 `{"code":"FORBIDDEN"}` не админу), `handlers.go` (`GetOnlineConfSession` → `200 {id, login}` из кеша; `SearchOnlineConfUsers` → `[{id, login}]` или `400 VALIDATION`), `api/` (тег `onlineconf`, `go generate ./internal/pkg/onlineconf/delivery/http/api/`);
- `docs/openapi.yaml`: тег `onlineconf`, пути `/onlineconf/admin/session` и `/onlineconf/admin/users`, схемы `OnlineConfError`, `OnlineConfUser`;
- `utils/httpjson.WriteError(w, status, code, message)` — JSON-ошибка для middleware;
- `cmd/main/main.go`: `onlineconfCtx` + `go adminCache.Run(onlineconfCtx)`; админские ручки на `adminRouter := apiRouter.NewRoute().Subrouter()` с `AdminOnly` (общие middleware `/api` тоже действуют); `onlineconfCancel()` при остановке.

## Принятые решения

- `/session` **не отдаёт `is_admin`**: доступ определяется кодом ответа (так решила ментор). Кнопка в профиле фронта показывается по `200` от `/session`.
- Миграции: `000008` — `is_admin` (Сейран), `000009` — таблицы OnlineConf (Лиза).
- Ручки `/keys` Лиза добавляет в тот же тег `onlineconf` и тот же пакет `onlineconf/delivery/http`. При конфликте в `api.gen.go` — перегенерировать.
- Коды ошибок OnlineConf: `FORBIDDEN`, `VALIDATION`, `NOT_FOUND`, `KEY_EXISTS`, `VERSION_CONFLICT`, `FILE_TOO_LARGE`, `FEATURE_DISABLED`; формат `{"code", "message"}`.

## Дальше (часть Сейрана, по ТЗ)

1. **Шаг 3** — пакет `internal/pkg/onlineconf`: функция вычисления флага (порядок: `enabled=false` → exclude → include → `fnv32a(key + ":" + user_id) % 100 < rollout_percent`) и клиент снимка (опрос `GetSnapshot(revision)` раз в 15 с, кеш в `atomic.Pointer`, запуск на `onlineconfCtx`). Зависит от `onlineconf.proto` Лизы — согласовать сообщения снимка; пока её нет, писать на интерфейсе и моке.
2. **Шаг 4** — `GET /api/onlineconf/runtime` (ключи скоупа `frontend` и `is_public`, флаги вычислены для текущего пользователя, формат в ТЗ). Не админская ручка — регистрировать на `apiRouter`, не на `adminRouter`.
3. **Шаг 5** — вложения: `internal/pkg/attachments/delivery/http/handlers.go` (`maxFileSize10MB`, `upload.ReadFile`): лимит из `attachments.max_file_size_bytes`, флаг `attachments.upload_enabled`, ответы `413 FILE_TOO_LARGE {limit_bytes}` и `403 FEATURE_DISABLED`. Значения по умолчанию: 10 МиБ, загрузка разрешена.
4. **Шаг 6** — деплой: сервис `onlineconf` в `docker-compose.yml` (`ONLINECONF_GRPC_ADDR=onlineconf:9004` для main), цель `onlineconf:9104` в `monitoring/prometheus.yml`, nginx `location /onlineconf/` с `try_files $uri /onlineconf/index.html`, первый админ `UPDATE users SET is_admin = true WHERE login = '...'`.
5. P1 `POST /keys/{id}/evaluate` (в main, тем же вычислением), P2 `/admins`, P3 RBAC.

## Окружение и подводные камни

- **Docker Hub с Mac Сейрана недоступен** (рвётся CDN). Сборка в обход: `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ~/bnn-build/<svc>/bnn ./cmd/<svc>`, `Dockerfile` из `FROM scratch` + `COPY bnn`, `docker build -t 2026_2_bnn-<backend|auth|notes> …`, `docker compose up -d --no-build`.
- GitHub и proxy.golang.org иногда рвутся — повторять команду.
- Генерация proto — по сервисам: `go generate ./internal/pkg/auth/delivery/grpc/` и `./internal/pkg/notes/delivery/grpc/` (gateway удалён). Моки: `go generate ./internal/pkg/<домен>/`.
- Покрытие: `make cover` (исключает `/mocks/`, `.gen.go`, `/gen/`). Требование РК — 60%; пакеты `mcp` и `onlineconf` пока без тестов.
- Сервер: `161.104.105.206` / `brandnewnotes.ru`, деплой GitHub Actions при push в `main`. В `.env` сервера обязательно `GRAFANA_ADMIN_PASSWORD` (иначе `docker compose up` падает), плюс `GRAFANA_ADMIN_USER`, `GRAFANA_ROOT_URL=https://brandnewnotes.ru/grafana/`.
- Локально есть тестовый аккаунт `test` (пароль у Сейрана). Админом его делать SQL выше, кеш обновится за 15 с.

## Вне хакатона (отложено)

- Личные API-токены для MCP (`bnn_…`, таблица `api_tokens`, миграция будет со следующим свободным номером) — спроектированы и проверены, в проект не внесены.
- Лиза: mTLS не подключён в коде, `grpcutil.NewServer(grpcMetrics ...)` с `[0]` переделать в обычный параметр, перевыпустить сертификаты. Старый коммит `38bc3ec` с ключами ещё доступен на GitHub по хешу — нужен запрос в GitHub Support.
