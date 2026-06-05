#!/usr/bin/env bash
set -e

cleanup() {
    echo ""
    echo "Stopping dev servers..."
    kill "$GO_PID" "$NEXT_PID" 2>/dev/null
    wait "$GO_PID" "$NEXT_PID" 2>/dev/null
    exit 0
}
trap cleanup INT TERM

# ── Освобождаем порты если заняты ───────────────────────────
for PORT in 8088 3000; do
    PIDS=$(lsof -ti :$PORT 2>/dev/null || true)
    if [ -n "$PIDS" ]; then
        echo "⚠️  Порт $PORT занят — освобождаю..."
        kill -9 $PIDS 2>/dev/null || true
        sleep 0.5
    fi
done

# ── Запуск Go бэкенда ────────────────────────────────────────
echo "Starting Go backend (port 8088)..."
go run -tags dev ./cmd/bot/ &
GO_PID=$!

# Ждём пока Go поднимется (макс 10 сек)
echo "Waiting for API..."
for i in $(seq 1 20); do
    if curl -sf http://localhost:8088/api/leagues > /dev/null 2>&1; then
        echo "✅ API готов"
        break
    fi
    if [ $i -eq 20 ]; then
        echo "❌ API не запустился за 10 секунд — проверь логи"
        echo "   Возможные причины:"
        echo "   - Ошибка компиляции Go"
        echo "   - Проблема с БД (проверь POSTGRES_DSN в .env)"
        echo "   - Порт 8088 уже занят другим процессом"
        kill "$GO_PID" 2>/dev/null
        exit 1
    fi
    sleep 0.5
done

# ── Запуск Next.js ───────────────────────────────────────────
echo "Starting Next.js frontend (port 3000)..."
cd web && npm run dev &
NEXT_PID=$!

echo ""
echo "  ✅ API:      http://localhost:8088"
echo "  ✅ Frontend: http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop both servers."

wait
