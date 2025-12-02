# Server Operations

## Запуск стека
Требуется Docker/Docker Compose.
```bash
docker compose -f docker-compose.dev.yml up -d --build
```
Проверка:
```bash
docker compose -f docker-compose.dev.yml ps
docker compose -f docker-compose.dev.yml logs -f ocpp-server sessions-service billing-service telemetry-service
```
Health:
```bash
curl http://localhost:8081/health      # ocpp-server
curl http://localhost:8082/health      # sessions-service
curl http://localhost:8083/health      # billing-service
curl http://localhost:8084/health      # telemetry-service
```

## Демонстрационный сценарий (OCPP)
1) Запусти эмулятор (Rust контейнер из compose или локальный Python GUI).
2) Подключи к `ws://localhost:8081/ocpp/ws?station_id=CS-XXX`.
3) В эмуляторе: Start session → MeterValues пойдут каждые ~2с → Stop session.
4) Проверки в БД:
```bash
docker exec -it dp-postgres psql -U postgres -d drivepower \
  -c "select id, station_id, connector_id, status, energy_kwh, transaction_id from charging_sessions order by id desc limit 10;"

docker exec -it dp-postgres psql -U postgres -d drivepower \
  -c "select * from telemetry_data order by id desc limit 10;"

docker exec -it dp-postgres psql -U postgres -d drivepower \
  -c "select * from billing_transactions order by id desc limit 10;"

docker exec -it dp-postgres psql -U postgres -d drivepower \
  -c "select * from tariffs;"
```
`tariffs` должен содержать хотя бы `Default` (сид добавлен), иначе биллинг будет пустым.

## Ручные запросы (если нужно)
- Остановить зависшую сессию напрямую:
```bash
curl -X POST http://localhost:8082/internal/ocpp/session-stop \
  -H "Content-Type: application/json" \
  -d '{"transaction_id":"<TX_ID>","end_time":"<ISO8601>","energy_kwh":5.0}'
```
- Просмотр станций:
```bash
docker exec -it dp-postgres psql -U postgres -d drivepower \
  -c "select id, vendor, model, status, last_heartbeat from charging_stations;"
```

## Пересборка отдельных сервисов
```bash
docker compose -f docker-compose.dev.yml build ocpp-server sessions-service billing-service telemetry-service
docker compose -f docker-compose.dev.yml up -d ocpp-server sessions-service billing-service telemetry-service
```
