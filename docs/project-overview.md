# Project Overview

## Structure
- **backend/services/**
  - **ocpp-server (Go)** — принимает OCPP‑кадры по WebSocket `/ocpp/ws`, ведёт учёт станций/коннекторов, дергает вспомогательные сервисы (sessions, billing, telemetry), пишет OCPP‑логи в Postgres.
  - **sessions-service (Go)** — хранит и завершает зарядные сессии (`charging_sessions`), кеширует активные сессии в Redis, отдаёт health.
  - **billing-service (Go)** — рассчитывает транзакции на основе энергии/тарифа (`billing_transactions`), каллбек `/internal/ocpp/session-stopped`.
  - **telemetry-service (Go)** — принимает MeterValues `/internal/ocpp/meter-values`, пишет в `telemetry_data`, поддерживает materialized view по энергии.
  - **auth-service (Go)** — базовая аутентификация/JWT (для полноты инфраструктуры).
  - **api-gateway (Go)** — проксирование API, health.
- **station-emulator (Rust)** — контейнерный эмулятор зарядной станции, шлёт OCPP‑кадры (Boot/Status/Start/Stop/MeterValues/Heartbeat).
- **station_simulator.py (Python + tkinter)** — локальный GUI‑эмулятор, управляет старт/стоп сессиями, отправляет MeterValues и Heartbeat.
- **csms/** — старая заглушка CSMS на Go (простейший WebSocket echo/health).
- **infra/** — вспомогательные конфиги; **docker-compose.dev.yml** запускает весь стек.
- **docs/** — документация (вы читаете её).

## Основная логика
1) Эмулятор (Rust контейнер или Python GUI) подключается к `ws://<host>:8081/ocpp/ws?station_id=<ID>` и отправляет кадры формата массива `[2,"<uid>","Action",{payload}]`.
2) **ocpp-server** парсит кадры, обновляет `charging_stations`/in‑memory state, дергает:
   - **sessions-service** `/internal/ocpp/session-start|stop` — создаёт/завершает `charging_sessions`, возвращает `session_id`.
   - **billing-service** `/internal/ocpp/session-stopped` — создаёт `billing_transactions` (использует тарифы из `tariffs`).
   - **telemetry-service** `/internal/ocpp/meter-values` — сохраняет телеметрию по `session_id`.
3) Postgres хранит все таблицы, Redis — кеш активных сессий.
4) API‑gateway и auth используются как вспомогательные компоненты (health/прокси).

## Технологии
- Go 1.23 (сборка на distroless), Postgres 15, Redis 7, Docker Compose.
- websockets (Python GUI), ws (Rust), gorilla/websocket (Go).
- tkinter для GUI.

## Логи (основное)
- **ocpp-server**: `info starting ocpp http server`; `station connected`; `sessions client returned non-success`/`billing client returned non-success` при ошибках HTTP 4xx/5xx; `meter values without session context` если нет session_id в txStore.
- **sessions-service**: `start session failed`/`stop session failed` с деталями SQL (FK, not found).
- **billing-service**: `failed to create billing transaction` если расчёт не прошёл.
- **telemetry-service**: `failed to store meter value` для невалидного payload.
- **эмулятор (Rust контейнер)**: отправленные/принятые кадры и ошибки WebSocket.
- **эмулятор (Python GUI)**: в окне Log — `Connecting`, `SEND: [...]`, `RECV: [...]`, ошибки соединения.
