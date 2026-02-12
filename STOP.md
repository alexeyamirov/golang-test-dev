# Быстрая остановка всех сервисов

## Способ 1: Использование скрипта (рекомендуется)

```powershell
.\stop-all.ps1
```

Скрипт автоматически:
- Остановит все запущенные Go приложения (api-gateway, data-ingestion, alert-processor, simulator)
- Остановит все Docker контейнеры

---

## Способ 2: Ручная остановка

### 1. Остановите приложения Go

В каждом терминале с запущенным сервисом нажмите `Ctrl+C`

Или остановите все процессы одной командой:
```powershell
Stop-Process -Name api-gateway,data-ingestion,alert-processor,simulator -Force -ErrorAction SilentlyContinue
```

### 2. Остановите Docker контейнеры

```powershell
docker-compose down
```

---

## Способ 3: Полная очистка (с удалением данных)

```powershell
# Остановить приложения
Stop-Process -Name api-gateway,data-ingestion,alert-processor,simulator -Force -ErrorAction SilentlyContinue

# Остановить и удалить все данные Docker
docker-compose down -v
```

⚠️ **Внимание:** Команда `down -v` удалит все данные из базы данных!

---

## Проверка, что все остановлено

```powershell
# Проверка процессов
Get-Process -Name api-gateway,data-ingestion,alert-processor,simulator -ErrorAction SilentlyContinue

# Проверка Docker контейнеров
docker ps

# Проверка портов
netstat -ano | findstr ":8080 :8081"
```

Если ничего не выводится - все остановлено.
