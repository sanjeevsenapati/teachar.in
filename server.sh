#!/usr/bin/env bash
# ==============================================================================
# TEACHAR.in - Server Process Management Script (Start / Stop / Restart / Status)
# ==============================================================================

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="teachar-server"
BINARY_PATH="${APP_DIR}/${BINARY_NAME}"
PUBLIC_PORT=8080
ADMIN_PORT=8081
PORT=8080
LOG_DIR="${APP_DIR}/logs"
LOG_FILE="${LOG_DIR}/app.log"

mkdir -p "${LOG_DIR}"

get_pid() {
    local pid
    pid=$(lsof -t -i :${PUBLIC_PORT} -sTCP:LISTEN 2>/dev/null)
    if [ -z "${pid}" ]; then
        pid=$(lsof -t -i :${ADMIN_PORT} -sTCP:LISTEN 2>/dev/null)
    fi
    if [ -z "${pid}" ]; then
        pid=$(pgrep -f "${BINARY_NAME}" | head -n 1)
    fi
    echo "${pid}"
}

build_app() {
    echo "🔨 Building Go application binary (${BINARY_NAME})..."
    cd "${APP_DIR}" || exit 1
    go build -o "${BINARY_NAME}" .
    if [ $? -ne 0 ]; then
        echo "❌ Build failed! Aborting start."
        exit 1
    fi
    echo "✅ Build completed successfully."
}

start_server() {
    local pid
    pid=$(get_pid)
    if [ -n "${pid}" ]; then
        echo "⚠️  TEACHAR server is already running (PID: ${pid})."
        return 0
    fi

    build_app

    echo "🚀 Starting TEACHAR Public App (:8080) & Private Admin Portal (:8081)..."
    nohup "${BINARY_PATH}" > "${LOG_FILE}" 2>&1 &
    sleep 2

    pid=$(get_pid)
    if [ -n "${pid}" ]; then
        echo "✅ TEACHAR dual-application server started successfully (PID: ${pid})."
        echo "🍵 Public Customer App:        https://localhost:${PUBLIC_PORT} (https://teachar.in:${PUBLIC_PORT})"
        echo "🔐 Private Staff/Admin Portal: https://localhost:${ADMIN_PORT} (https://admin.teachar.in:${ADMIN_PORT})"
        echo "📜 Logs: tail -f ${LOG_FILE}"
    else
        echo "❌ Failed to start server! Check logs in ${LOG_FILE}."
    fi
}

stop_server() {
    local pid
    pid=$(get_pid)
    if [ -z "${pid}" ]; then
        echo "ℹ️  TEACHAR server is not running."
        return 0
    fi

    echo "🛑 Stopping TEACHAR servers (PID: ${pid})..."
    kill "${pid}" 2>/dev/null

    local count=0
    while [ ${count} -lt 10 ]; do
        if ! kill -0 "${pid}" 2>/dev/null; then
            echo "✅ Servers stopped cleanly."
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done

    echo "⚠️  Force killing server (PID: ${pid})..."
    kill -9 "${pid}" 2>/dev/null
    echo "✅ Server killed."
}

restart_server() {
    echo "🔄 Restarting TEACHAR dual-application server..."
    stop_server
    sleep 1
    start_server
}

status_server() {
    local pid
    pid=$(get_pid)
    if [ -n "${pid}" ]; then
        echo "🟢 TEACHAR server is RUNNING (PID: ${pid})."
        echo "🍵 Public Customer App:        https://localhost:${PUBLIC_PORT} (https://teachar.in:${PUBLIC_PORT})"
        echo "🔐 Private Staff/Admin Portal: https://localhost:${ADMIN_PORT} (https://admin.teachar.in:${ADMIN_PORT})"
        echo "📜 Log File: ${LOG_FILE}"
    else
        echo "🔴 TEACHAR server is STOPPED."
    fi
}

case "$1" in
    start)
        start_server
        ;;
    stop)
        stop_server
        ;;
    restart)
        restart_server
        ;;
    status)
        status_server
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac
