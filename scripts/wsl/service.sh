#!/usr/bin/env bash
set -euo pipefail

CMD="${1:-status}"

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    SUDO="sudo"
fi

case "${CMD}" in
    status)
        ${SUDO} systemctl status boyan --no-pager
        ;;
    logs)
        ${SUDO} journalctl -u boyan -f
        ;;
    restart)
        echo "Restarting boyan service..."
        ${SUDO} systemctl restart boyan
        ${SUDO} systemctl status boyan --no-pager
        ;;
    stop)
        echo "Stopping boyan service..."
        ${SUDO} systemctl stop boyan
        ;;
    start)
        echo "Starting boyan service..."
        ${SUDO} systemctl start boyan
        ${SUDO} systemctl status boyan --no-pager
        ;;
    *)
        echo "Usage: $0 {status|logs|restart|stop|start}"
        exit 1
        ;;
esac
