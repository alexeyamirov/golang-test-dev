#!/bin/bash
# Тестирование API TR181 Cloud Platform (Linux и macOS)
# Использование: ./test-api.sh

echo "=== Testing TR181 Cloud Platform API ==="
echo ""

# Ждём накопления данных
echo "Waiting 5 seconds for data to accumulate..."
sleep 5

# 1. Health check
echo "1. Health Check:"
if curl -sf http://localhost:8080/health > /dev/null; then
    echo "   ✓ API Gateway: ok"
else
    echo "   ✗ API Gateway: not responding"
fi
echo ""

# 2. Метрики
echo "2. Testing Metrics API:"
SERIAL="DEV-00000001"
FROM="2024-01-01T00:00:00Z"
TO="2025-12-31T23:59:59Z"

for metric in cpu-usage memory-usage wifi-2ghz-signal cpu-temperature; do
    url="http://localhost:8080/api/v1/metric/$metric?serial-number=$SERIAL&from=$FROM&to=$TO"
    response=$(curl -sf "$url" 2>/dev/null || echo "[]")
    count=$(echo "$response" | grep -o '"value"' | wc -l)
    if [ "$count" -gt 0 ]; then
        echo "   ✓ $metric: found data"
    else
        echo "   ⚠ $metric: no data yet (wait longer)"
    fi
done
echo ""

# 3. Алерты
echo "3. Testing Alerts API:"
for alert in high-cpu-usage low-wifi; do
    url="http://localhost:8080/api/v1/alert/$alert?serial-number=$SERIAL&from=$FROM&to=$TO"
    if curl -sf "$url" > /dev/null; then
        echo "   ✓ $alert: OK"
    else
        echo "   ✗ $alert: error"
    fi
done
echo ""

# 4. gRPC (если grpcurl установлен)
if command -v grpcurl &> /dev/null; then
    echo "4. Testing gRPC:"
    if grpcurl -plaintext -d '{"metric_type":"cpu-usage","serial_number":"DEV-00000001","from":0,"to":0}' localhost:9090 tr181.api.TR181Api/GetMetric 2>/dev/null | head -1; then
        echo "   ✓ gRPC: OK"
    else
        echo "   ⚠ gRPC: skipped or failed"
    fi
    echo ""
fi

echo "=== Testing Complete ==="
echo ""
echo "Note: If no data, wait 30-60 seconds for simulator to send data."
echo ""
