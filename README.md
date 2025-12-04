# DrivePower

Платформа для управления зарядными станциями: учёт сессий, телеметрии, биллинга и единая точка входа для клиентов (мобильное/веб).

## Содержание
- [Цели и функционал](#цели-и-функционал)
- [Общая архитектура](#общая-архитектура)
- [Сервисы и обязанности](#сервисы-и-обязанности)
- [Основные потоки](#основные-потоки)
- [Стек и зависимости](#стек-и-зависимости)
- [Структура репозитория](#структура-репозитория)
- [Конфигурация и ENV](#конфигурация-и-env)
- [Быстрый старт (dev)](#быстрый-старт-dev)
- [Миграции БД](#миграции-бд)
- [Запуск сервисов вручную (go run)](#запуск-сервисов-вручную-go-run)
- [API Gateway: сценарий клиента](#api-gateway-сценарий-клиента)
- [Эмулятор станции](#эмулятор-станции)
- [Проверка и отладка](#проверка-и-отладка)
- [Планы и ограничения](#планы-и-ограничения)
- [Скриншоты](#скриншоты)

## Цели и функционал
- Базовая аутентификация (signup/login, JWT).
- Приём OCPP-сообщений от станций (Boot/Status/Start/StopTransaction, MeterValues).
- Учёт сессий, активные сессии в Redis, история по пользователю.
- Телеметрия и суммарная энергия по сессии.
- Биллинг: расчёт amount = energy_kwh * price_per_kwh по активному тарифу.
- Единая внешняя точка — API Gateway с JWT-мидлварой.
- Эмулятор станции для end-to-end проверки.

## Общая архитектура
- **API Gateway** — внешний вход для клиентов; JWT-проверка; прокси в auth/sessions/billing/stations.
- **Auth-service** — регистрация, логин, выдача JWT (HS256), хранение пользователей (Postgres).
- **OCPP-server** — WebSocket `/ocpp/ws?station_id=...`; обрабатывает BootNotification, StatusNotification, Start/StopTransaction; пишет логи; дергает sessions/billing/telemetry.
- **Sessions-service** — хранение сессий (Postgres), кэш активных (Redis), история пользователя.
- **Telemetry-service** — приём MeterValues, хранение в `telemetry_data`, суммирование энергии.
- **Billing-service** — берёт активный тариф, считает сумму, пишет транзакцию.
- **Postgres/Redis** — общие зависимости.
- **Симулятор станции** — отправляет OCPP-события для тестов.

## Сервисы и обязанности
- **auth-service**
  - `POST /auth/signup`, `POST /auth/login`, `GET /health`.
  - Таблица `users`. JWT-клеймы: `user_id`, `role`, `iat`, `exp`.
- **ocpp-server**
  - WebSocket `/ocpp/ws?station_id=...`.
  - Поддержка: BootNotification, StatusNotification, StartTransaction, StopTransaction.
  - Логирование OCPP в Postgres, вызовы sessions/billing/telemetry.
- **sessions-service**
  - `POST /internal/ocpp/session-start`, `POST /internal/ocpp/session-stop`.
  - `GET /sessions/me`, `GET /sessions/active`.
  - Хранение в Postgres; активные сессии в Redis.
- **telemetry-service**
  - `POST /internal/ocpp/meter-values`.
  - Таблица `telemetry_data`; сумма энергии по session_id (view или MAX-MIN).
- **billing-service**
  - `POST /internal/ocpp/session-stopped`.
  - `GET /billing/me/transactions`.
  - Тарифы в `tariffs`, транзакции в `billing_transactions`.
- **api-gateway**
  - Внешние маршруты: `/api/auth/signup`, `/api/auth/login`, `/api/sessions/me`, `/api/billing/me/transactions`, `/api/stations`.
  - JWT-мидлварь (HS256), проверка секрета, извлечение `user_id`.

## Основные потоки
1. **Клиент**: signup → login → получает JWT → ходит в API Gateway (`/api/sessions/me`, `/api/billing/me/transactions`, `/api/stations`).
2. **Станция (OCPP)**: подключение к `/ocpp/ws`; BootNotification → Accepted → Status → StartTransaction → StopTransaction → лог в Postgres → вызовы sessions/billing/telemetry.
3. **Биллинг**: при StopTransaction биллинг получает `session_id`, `user_id`, `energy_kwh`; берёт активный тариф, считает `amount`, пишет транзакцию.

## Стек и зависимости
- Go 1.21, Postgres 15, Redis 7, Docker Compose.
- JWT (HS256), bcrypt.
- Логирование: zap. Конфигурации: env/YAML (`CONFIG_FILE`).
- WebSocket (gorilla/websocket) для OCPP.

## Структура репозитория
```
backend/
  libs/                #: logging, config, db, redis
  services/
    auth-service/
    ocpp-server/
    sessions-service/
    telemetry-service/
    billing-service/
    api-gateway/
infra/
docs/
station_simulator.py   # эмулятор станции
docker-compose.dev.yml # dev-стек (postgres, redis, сервисы)
go.mod / go.sum
```

## Конфигурация и ENV
Общее: `LOG_LEVEL`, `CONFIG_FILE` (опционально).  
\* — обязательные.

- **Auth**: `AUTH_POSTGRES_DSN`*, `AUTH_HTTP_PORT` (8080+), `AUTH_JWT_SECRET`*, `AUTH_JWT_EXPIRES_MINUTES` (60).
- **Sessions**: `SESSIONS_POSTGRES_DSN`*, `SESSIONS_HTTP_PORT`, `SESSIONS_REDIS_ADDR`, `SESSIONS_REDIS_PASSWORD`, `SESSIONS_REDIS_DB`, `SESSIONS_REDIS_TTL`.
- **Telemetry**: `TELEMETRY_POSTGRES_DSN`*, `TELEMETRY_HTTP_PORT`.
- **Billing**: `BILLING_POSTGRES_DSN`*, `BILLING_HTTP_PORT`.
- **OCPP**: `OCPP_POSTGRES_DSN`*, `OCPP_HTTP_PORT`, `SESSIONS_SERVICE_URL`, `BILLING_SERVICE_URL`, `TELEMETRY_SERVICE_URL`.
- **API Gateway**: `API_GATEWAY_HTTP_PORT`, `API_GATEWAY_JWT_SECRET`* (тот же, что в auth), `AUTH_SERVICE_URL`, `SESSIONS_SERVICE_URL`, `BILLING_SERVICE_URL`, `STATIONS_SERVICE_URL`.

## Быстрый старт (dev)
1. Поднять инфраструктуру:
   ```bash
   docker-compose -f docker-compose.dev.yml up -d postgres redis
   ```
2. Создать базы (если ещё не созданы): `drivepower_auth`, `drivepower_ocpp`, `drivepower_sessions`, `drivepower_telemetry`, `drivepower_billing`.
3. Применить миграции:
   ```bash
   docker-compose -f docker-compose.dev.yml exec -T postgres psql -U postgres -d drivepower_auth      < backend/services/auth-service/migrations/0001_create_users_table.sql
   docker-compose -f docker-compose.dev.yml exec -T postgres psql -U postgres -d drivepower_ocpp      < backend/services/ocpp-server/migrations/0001_init.sql
   docker-compose -f docker-compose.dev.yml exec -T postgres psql -U postgres -d drivepower_sessions  < backend/services/sessions-service/migrations/0001_create_charging_sessions.sql
   docker-compose -f docker-compose.dev.yml exec -T postgres psql -U postgres -d drivepower_telemetry < backend/services/telemetry-service/migrations/0001_create_telemetry_data.sql
   docker-compose -f docker-compose.dev.yml exec -T postgres psql -U postgres -d drivepower_billing   < backend/services/billing-service/migrations/0001_init_billing.sql
   ```
4. Скачать зависимости: `go mod tidy` (создаст `go.sum`).
5. Запустить сервисы (каждый в своём терминале) с нужными ENV:
   ```bash
   AUTH_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/drivepower_auth?sslmode=disable" \
   AUTH_HTTP_PORT="8080" AUTH_JWT_SECRET="dev-secret" \
   go run ./backend/services/auth-service/cmd/auth-service

   SESSIONS_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/drivepower_sessions?sslmode=disable" \
   SESSIONS_HTTP_PORT="8082" SESSIONS_REDIS_ADDR="localhost:6379" \
   go run ./backend/services/sessions-service/cmd/sessions-service

   TELEMETRY_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/drivepower_telemetry?sslmode=disable" \
   TELEMETRY_HTTP_PORT="8084" \
   go run ./backend/services/telemetry-service/cmd/telemetry-service

   BILLING_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/drivepower_billing?sslmode=disable" \
   BILLING_HTTP_PORT="8083" \
   go run ./backend/services/billing-service/cmd/billing-service

   OCPP_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/drivepower_ocpp?sslmode=disable" \
   OCPP_HTTP_PORT="8081" \
   SESSIONS_SERVICE_URL="http://localhost:8082" \
   BILLING_SERVICE_URL="http://localhost:8083" \
   TELEMETRY_SERVICE_URL="http://localhost:8084" \
   go run ./backend/services/ocpp-server/cmd/ocpp-server

   API_GATEWAY_HTTP_PORT="8090" \
   API_GATEWAY_JWT_SECRET="dev-secret" \
   AUTH_SERVICE_URL="http://localhost:8080" \
   SESSIONS_SERVICE_URL="http://localhost:8082" \
   BILLING_SERVICE_URL="http://localhost:8083" \
   STATIONS_SERVICE_URL="http://localhost:8081" \
   go run ./backend/services/api-gateway/cmd/api-gateway
   ```
   Или собрать/поднять через `docker-compose.dev.yml`, добавив сервисы в файл.

## Миграции БД
- Auth: `backend/services/auth-service/migrations/0001_create_users_table.sql`
- OCPP: `backend/services/ocpp-server/migrations/0001_init.sql`
- Sessions: `backend/services/sessions-service/migrations/0001_create_charging_sessions.sql`
- Telemetry: `backend/services/telemetry-service/migrations/0001_create_telemetry_data.sql`
- Billing: `backend/services/billing-service/migrations/0001_init_billing.sql`

## Запуск сервисов вручную (go run)
- Каждый сервис — отдельный `cmd/.../main.go`.
- Перед запуском: задать ENV (DSN, порты, JWT, URL других сервисов).
- Проверка: `curl http://localhost:<port>/health`.

## API Gateway: сценарий клиента
1. `POST /api/auth/signup` → создать пользователя.
2. `POST /api/auth/login` → получить JWT.
3. С JWT (Authorization: Bearer ...):
   - `GET /api/sessions/me` — история сессий.
   - `GET /api/billing/me/transactions` — биллинг.
   - `GET /api/stations` — статусы станций.
4. Для e2e: запустить эмулятор станции, после Start/StopTransaction данные появятся в `/api/sessions/me` и `/api/billing/me/transactions`.

## Эмулятор станции
- Файл: `station_simulator.py`.
- Назначение: эмулировать зарядную станцию и слать OCPP-сообщения (Boot/Status/Start/StopTransaction, MeterValues) на WebSocket OCPP-сервера.
- Подготовка:
  - Убедитесь, что OCPP-server запущен и доступен по `ws://localhost:8081/ocpp/ws?station_id=...`.
  - Настройте параметры подключения в коде/ENV (если предусмотрены).
- Запуск (пример):
  ```bash
  python station_simulator.py --url ws://localhost:8081/ocpp/ws --station-id STATION-001
  ```
- После запуска: в логах OCPP-сервера видно Boot/Status/Start/Stop; в sessions/billing появляются записи, проверяйте через API Gateway.

## Проверка и отладка
- Health: `/health` у каждого сервиса.
- Логи: stdout сервиса или `docker compose logs -f <service>`.
- Типичные ошибки:
  - Нет `go.sum` → `go mod tidy`.
  - Не подхватились ENV → запуск в одной строке без пробелов после `\` или через `export`.
  - JWT mismatch → убедитесь, что Gateway и Auth используют один секрет.

## Планы и ограничения
- MVP: только базовые OCPP-команды, простой тариф, нет refresh-токенов, нет продвинутой аналитики.
- Возможные доработки: RemoteStart/Stop, сложные тарифы, интеграция платежей, наблюдаемость (metrics/tracing), UI.

## Скриншоты
- Архитектурная схема: ![Architecture](docs/screenshots/architecture.png)
- Подключение эмулятора: ![emulator](docs/screenshots/emulator.png)
- Список таблиц в БД: ![table](docs/screenshots/table.png)
- Таблица billing_transactions: ![billing_transactions](docs/screenshots/billing_transactions.png)
- Таблица charging_sessions: ![charging_sessions](docs/screenshots/charging_sessions.png)
- Таблица charging_stations: ![charging_stations](docs/screenshots/charging_stations.png)
- Таблица ocpp_messages: ![ocpp_messages](docs/screenshots/ocpp_messages.png)
- Таблица stations: ![stations](docs/screenshots/stations.png)
- Таблица tariffs: ![tariffs](docs/screenshots/tariffs.png)
- Таблица telemetry_data: ![telemetry_data](docs/screenshots/telemetry_data.png)
- Таблица users: ![users](docs/screenshots/users.png)

## Запросы для просмотра содержимого таблиц
-- Биллинг
docker exec -it dp-postgres psql -U postgres -d drivepower -c "SELECT * FROM billing_transactions ORDER BY id DESC LIMIT 50;"

-- Сессии
docker exec -it dp-postgres psql -U postgres -d drivepower -c "SELECT * FROM charging_sessions ORDER BY id DESC LIMIT 50;"

-- Статусы станций (регистрация/heartbeat)
docker exec -it dp-postgres psql -U postgres -d drivepower -c "SELECT * FROM charging_stations ORDER BY id DESC LIMIT 50;"

-- Сырой обмен OCPP (если логируете)
docker exec -it dp-postgres psql -U postgres -d drivepower -c "SELECT * FROM ocpp_messages ORDER BY id DESC LIMIT 50;"

-- Справочник станций (если есть)
docker exec -it dp-postgres psql -U postgres -d drivepower -c "SELECT * FROM stations ORDER BY id DESC LIMIT 50;"

-- Тарифы
docker exec -it dp-postgres psql -U postgres -d drivepower -c "SELECT * FROM tariffs ORDER BY id DESC LIMIT 50;"

-- Телеметрия (MeterValues)
docker exec -it dp-postgres psql -U postgres -d drivepower -c "SELECT * FROM telemetry_data ORDER BY id DESC LIMIT 50;"

-- Пользователи
docker exec -it dp-postgres psql -U postgres -d drivepower -c "SELECT * FROM users ORDER BY id DESC LIMIT 50;"