#!/usr/bin/env bash
set -Eeuo pipefail

[[ "${EUID}" -eq 0 ]] || { echo "Run as root." >&2; exit 1; }
systemctl disable --now xpanel-api xpanel-worker 2>/dev/null || true
rm -f /etc/systemd/system/xpanel-api.service /etc/systemd/system/xpanel-worker.service
rm -f /etc/nginx/sites-enabled/xpanel /etc/nginx/sites-available/xpanel
systemctl daemon-reload
systemctl reload nginx 2>/dev/null || true
echo "Services and web entry removed. Database and /etc/xpanel were retained."

