# DrivePower

Разработка ПО для зарядных станций электрокаров.

## Сервисы

- **CSMS (Go)** — минимальный OCPP 2.0.x сервер, который принимает WebSocket-подключения на `/ocpp/`, отвечает на `BootNotification`/`Heartbeat`, задаёт интервал пинга 10 секунд, закрывает сеанс после тестового heartbeat и сохраняет данные станции в PostgreSQL (BootNotification — upsert, Heartbeat — обновление `last_seen_at`). HTTP-эндпоинт `/health` используется Docker Compose для проверки готовности.
- **station-emulator (Rust)** — эмулятор зарядной станции. Берёт настройки из переменных окружения (можно задать через `.env`), подключается к CSMS, проходит цикл `BootNotification → Heartbeat`, затем корректно завершает соединение после того как сервер отправит закрытие.
- **PostgreSQL** — база данных, в которой хранятся сведения о зарядных станциях. Контейнер стартует автоматически вместе с Compose, миграции применяются отдельным сервисом `migrate`.

Go-сервис использует библиотеку `github.com/gorilla/websocket`, которая зафиксирована в `csms/third_party/github.com/gorilla/websocket` и подключается через директиву `replace` — благодаря этому сборка не обращается к интернету.

## Быстрый старт через Docker Compose

1. Установите Docker Desktop или Docker Engine + Docker Compose Plugin.
2. Склонируйте репозиторий и перейдите в его директорию.
3. Проверьте файл `emulator.env`: значение `CSMS_URL` должно указывать на `ws://csms:8080/ocpp` (без завершающего `/`). При необходимости отредактируйте остальные параметры станции.
4. Выполните сборку и запуск всех сервисов одной командой:

   ```bash
   docker compose up --build
   ```

   Compose соберёт образы, поднимет PostgreSQL, применит миграции, запустит CSMS (порт 8080) и после успешного healthcheck стартует эмулятор.

5. Наблюдайте логи прямо в терминале Compose. В них будут видно:
   - на стороне CSMS — апгрейд WebSocket, входящие фреймы, записи об обновлении БД, ответы и сообщение о закрытии после heartbeat;
   - на стороне эмулятора — формирование адреса, отправка BootNotification/Heartbeat и обработка закрытия.

6. Для остановки сервисов нажмите `Ctrl+C` в терминале с Compose. Чтобы удалить контейнеры, выполните `docker compose down`.

## Запуск в VS Code

1. Откройте папку репозитория в VS Code.
2. В терминале №1 запустите `docker compose up --build`. Это поднимет оба сервиса и позволит смотреть объединённые логи прямо в окне редактора.
3. Если хотите разнести логи по отдельным окнам, после успешной сборки выполните `docker compose logs -f csms` и `docker compose logs -f emulator` в отдельных терминалах VS Code.
4. Для ручной перезагрузки эмулятора выполните `docker compose restart emulator`; чтобы перезапустить всё — `docker compose down && docker compose up`.

## Отладка без Docker

1. Убедитесь, что в системе установлен Go >= 1.21 и Rust toolchain.
2. Поднимите PostgreSQL (локально или в Docker) и примените миграции из каталога `csms/migrations` (например, через [golang-migrate](https://github.com/golang-migrate/migrate)).
3. В каталоге `csms` выполните `go build` и запустите бинарь с переменной окружения `DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable`.
4. В каталоге `station-emulator` выполните `cargo run` (предварительно создайте `.env` или задайте переменные окружения).

## Работа с базой данных и миграциями

- Все миграции лежат в `csms/migrations`. При запуске `docker compose up --build` сервис `migrate` автоматически выполняет `migrate -path /migrations -database <DSN> up`.
- Для повторного применения миграций вручную выполните:

  ```bash
  docker compose run --rm migrate up
  ```

- Чтобы посмотреть содержимое таблицы `stations` после тестового сценария:

  ```bash
  docker compose exec postgres psql -U csms -d csms -c "TABLE stations"
  ```

- Для очистки состояния базы данных выполните `docker compose down -v` (удалит volume `postgres-data`).

## Решение конфликтов слияния

Если при слиянии веток GitHub показывает конфликты в `csms/go.mod`, `csms/go.sum` или `csms/main.go`, выполните локально:

```bash
git checkout <ваша-ветка>
git fetch origin
git merge origin/main
# вручную поправьте конфликтующие файлы, ориентируясь на текущие изменения в этом коммите
git add csms/go.mod csms/go.sum csms/main.go
git commit
git push
```

После разрешения конфликтов повторите операцию Merge Request.
