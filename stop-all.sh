#!/bin/bash
# Быстрая остановка всех сервисов (Linux и macOS)
# Вызывает scripts/stop-all.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "$SCRIPT_DIR/scripts/stop-all.sh"
