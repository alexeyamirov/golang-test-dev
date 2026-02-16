# Time Series базы данных

Метрики TR181 — это временные ряды: `(serial_number, metric_type, value, timestamp)`.

## Текущее решение: TimescaleDB

Сейчас используется **PostgreSQL + TimescaleDB** — расширение, превращающее PostgreSQL в time series БД:

- Hypertables с партиционированием по времени
- Сжатие старых данных
- Оптимизированные запросы по временным диапазонам
- SQL и экосистема Postgres

Для многих сценариев этого достаточно.

---

## Альтернативы

### InfluxDB
- Заточен под time series
- Высокая скорость записи
- Свой язык запросов (InfluxQL, Flux)
- Один репозиторий — InfluxData

### QuestDB
- SQL поверх time series
- Очень быстрая запись и аналитика
- Хорошая поддержка сжатия

### VictoriaMetrics
- Совместимость с Prometheus
- Низкое потребление ресурсов
- Подходит для метрик мониторинга

### ClickHouse
- Колоночная СУБД
- Сильная аналитика
- Сложнее в эксплуатации

---

## Миграция с PostgreSQL

Для перехода на InfluxDB/QuestDB потребуется:

1. Новый writer (адаптер) в `storage` вместо `database.PostgresDB`
2. Новые схемы/бакеты под time series
3. Обновление API Gateway для чтения из новой БД

TimescaleDB остаётся разумным выбором, если хочется остаться в мире PostgreSQL.
