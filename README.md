# DrivePower

Полноценный учебный стенд для отработки взаимодействия зарядной станции с CSMS по протоколу OCPP 2.0.x. Репозиторий содержит сервер, эмулятор станции, миграции PostgreSQL и инфраструктуру для запуска всего комплекта в Docker Compose.

## Основные компоненты

| Компонент | Язык | Назначение |
|-----------|------|------------|
| [`csms`](./csms) | Go | Минимальный CSMS, принимающий WebSocket-подключения, обрабатывающий сообщения `BootNotification`, `Heartbeat` и `StatusNotification`, а также сохраняющий сведения о станции и коннекторах в PostgreSQL. |
| [`station-emulator`](./station-emulator) | Rust | CLI-эмулятор зарядной станции, повторяющий базовый цикл регистрации и отправки статусов в CSMS. |
| [`docker-compose.yml`](./docker-compose.yml) | YAML | Описывает инфраструктуру: PostgreSQL, сервис миграций, CSMS и эмулятор станции. |

Дополнительно в каталоге [`csms/migrations`](./csms/migrations) лежат миграции базы данных, а в [`csms/internal`](./csms/internal) — код хранилища, реестра статусов и простого WebSocket-реализации, используемой для тестов.

## Быстрый старт через Docker Compose

1. Установите Docker Engine и Docker Compose Plugin (или Docker Desktop).
2. Склонируйте репозиторий и перейдите в каталог проекта:

   ```bash
   git clone https://github.com/<org>/DrivePower.git
   cd DrivePower
   ```

3. Проверьте файл [`emulator.env`](./emulator.env). По умолчанию `CSMS_URL=ws://csms:8080/ocpp`, что подходит для запуска через Compose. При необходимости отредактируйте идентификатор станции, адреса коннекторов и т. д.
4. Запустите окружение:

   ```bash
   docker compose up --build
   ```

   Команда соберёт образы, поднимет PostgreSQL, применит миграции и запустит сервисы. После прохождения healthcheck эмулятор подключится к CSMS и начнёт обмен сообщениями.
5. Чтобы остановить окружение, нажмите `Ctrl+C`. Для очистки контейнеров и volume выполните `docker compose down -v`.

### Что увидите в логах

- **CSMS**: соединения WebSocket, содержимое входящих/исходящих фреймов, SQL-операции (успех/ошибка), уведомления о состоянии коннекторов.
- **Эмулятор**: шаги подключения, формирование сообщений, ответы от сервера, обновления статуса.
- **Миграции**: применение скриптов к PostgreSQL при запуске.

## Сервис CSMS (Go)

### Конфигурация

CSMS использует одну переменную окружения:

- `DATABASE_URL` — строка подключения PostgreSQL в формате `postgres://user:pass@host:5432/dbname?sslmode=disable`.

При запуске через Compose значение задаётся автоматически. Для локального запуска экспортируйте переменную вручную:

```bash
export DATABASE_URL=postgres://csms:csms@localhost:5432/csms?sslmode=disable
cd csms
go run ./...
```

### Точки входа и протокол

- HTTP-эндпоинт `/health` — используется Compose для проверки готовности сервиса.
- WebSocket-эндпоинт `/ocpp/{stationId}` — точка подключения зарядных станций. Сервер требует сабпротокол `ocpp2.0`.

Последовательность обмена с эмулятором:

1. `BootNotification` — сервер отвечает `Accepted`, фиксирует интервал heartbeat (10 секунд) и сохраняет сведения о станции.
2. `Heartbeat` — сервер подтверждает, обновляет `last_seen_at` и **оставляет соединение открытым** для последующих сообщений.
3. `StatusNotification` — сервер валидирует полезную нагрузку, обновляет in-memory реестр и БД, публикует событие в канал `statusEvents`.

Все сообщения и итоги обработки выводятся в стандартный лог Go.

### Работа с базой данных

Код хранилища находится в [`csms/internal/storage`](./csms/internal/storage). Репозиторий `PostgresStationRepository` выполняет три основные операции:

- Upsert BootNotification в таблицу `stations`.
- Обновление `last_seen_at` станции.
- Upsert статусов коннекторов в таблицу `station_connector_statuses`.

Перед выполнением каждого SQL-запроса в лог выводится подробное описание, а после — сообщение об успешной записи или ошибке. Логи помогают отслеживать миграции и фактические операции записи в PostgreSQL.

### Реестр статусов

Модуль [`csms/internal/registry`](./csms/internal/registry) поддерживает состояние коннекторов в памяти. Каждое обновление формирует событие `StatusEvent`, которое можно перенаправить на внешние потребители. По умолчанию события просто логируются.

## Эмулятор станции (Rust)

Каталог [`station-emulator`](./station-emulator) содержит небольшое приложение на Rust, моделирующее поведение зарядной станции.

- Конфигурация считывается из `.env` или переменных окружения (см. [`emulator.env`](./emulator.env)).
- После запуска эмулятор подключается к CSMS, выполняет `BootNotification`, периодически отправляет `Heartbeat`, а также может слать `StatusNotification` в зависимости от настроек.
- Сообщения и ответы сервера выводятся в stdout.

Для локального запуска без Docker:

```bash
cd station-emulator
cargo run
```

## Миграции базы данных

Миграции находятся в [`csms/migrations`](./csms/migrations) и совместимы с инструментом [golang-migrate](https://github.com/golang-migrate/migrate).

- Применение в Docker: автоматически выполняется сервисом `migrate` (команда `migrate -path /migrations -database <DSN> up`).
- Ручной запуск:

  ```bash
  docker compose run --rm migrate up
  ```

- Откат последнего шага:

  ```bash
  docker compose run --rm migrate down 1
  ```

## Локальная разработка без Docker

1. Убедитесь, что установлены Go ≥ 1.21, Rust toolchain и PostgreSQL.
2. Примените миграции к локальной БД `csms`.
3. Запустите сервер:

   ```bash
   cd csms
   DATABASE_URL=postgres://csms:csms@localhost:5432/csms?sslmode=disable go run ./...
   ```

4. В другом терминале запустите эмулятор (см. выше).

Для отладки WebSocket можно использовать `wscat`:

```bash
wscat -c ws://localhost:8080/ocpp/STATION-1 -H "Sec-WebSocket-Protocol: ocpp2.0"
```

## Структура каталогов

```
DrivePower/
├── csms/                    # Go-сервис CSMS
│   ├── internal/            # Пакеты хранилища, реестра, простого ws
│   ├── migrations/          # SQL-миграции PostgreSQL
│   ├── third_party/         # Зафиксированные внешние зависимости
│   └── main.go              # Точка входа сервера
├── station-emulator/        # Эмулятор станции (Rust)
├── docker-compose.yml       # Инфраструктура стенда
├── emulator.env             # Пример настроек эмулятора
└── README.md                # Документация (вы читаете её)
```

## Полезные команды

- Сборка и тесты CSMS: `cd csms && go test ./...`
- Форматирование Go-кода: `cd csms && gofmt -w <files>`
- Просмотр данных в БД: `docker compose exec postgres psql -U csms -d csms -c "TABLE stations"`
- Прослушивание логов отдельных сервисов: `docker compose logs -f csms`, `docker compose logs -f emulator`

## Лицензия

Проект предназначен для учебных целей. Используйте и модифицируйте свободно в рамках внутренних проектов.
