# Emulator Operations

## Вариант 1: контейнерный эмулятор (Rust, в Compose)
Уже включён в `docker-compose.dev.yml` как `station-emulator`. Он шлёт Boot/Status/Start/Stop/Heartbeat/MeterValues автоматически. Запуск стека:
```bash
docker compose -f docker-compose.dev.yml up -d station-emulator
```
Логи:
```bash
docker compose -f docker-compose.dev.yml logs -f station-emulator
```
URL для подключения задан в compose (`CSMS_URL=ws://ocpp-server:8081/ocpp/ws`), `station_id` берётся из query.

## Вариант 2: локальный GUI-эмулятор (Python + tkinter)

### Подготовка
1) Создай/активируй venv (если ещё нет):
```bash
python3 -m venv .venv
source .venv/bin/activate
```
2) Установи зависимости:
```bash
pip install websockets
```

### Запуск
```bash
python station_simulator.py
```

### Использование
- Поле CSMS URL: укажи `ws://localhost:8081/ocpp/ws?station_id=CS-001` (или другой station_id).
- Station ID в форме не переписывает URL, если в URL уже есть station_id — меняй параметр прямо в URL перед Connect.
- Выбери число коннекторов (1–4), при необходимости включи Auto-start.
- Нажми **Connect**. В логе увидишь Connected и Boot/Status.
- **Start session** на коннекторе: устанавливает статус Charging, генерирует transactionId, шлёт StartTransaction, запускает прирост энергии и MeterValues каждые ~2с.
- **Stop session**: шлёт StopTransaction, останавливает прирост, возвращает статус Available.
- Heartbeat: уходит раз в 30с (можно менять).
- Log: показывает SEND/RECV/ошибки.

### Полезные советы
- Для проверки разных станций ставь нужный `station_id` в URL и переподключайся (Disconnect → Connect).
- Если нужно принудительно завершить зависшую сессию — используй REST в sessions-service (см. server-operations.md).
- MeterValues появляются только при активной сессии и наличии transactionId; ocpp-server сохраняет их в `telemetry_data` если session_id известен.
