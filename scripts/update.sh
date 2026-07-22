#!/usr/bin/env bash
set -Eeuo pipefail

DOMAIN="${DOMAIN:-x.unsvalo.com}"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

[[ "${EUID}" -eq 0 ]] || { echo "Run as root." >&2; exit 1; }
cd "$SOURCE_DIR"
if [[ -n "$(git status --porcelain)" ]]; then
  echo "Source tree contains local changes; update stopped." >&2
  exit 1
fi

git pull --ff-only
export PATH="/usr/local/go/bin:/usr/local/node/bin:${PATH}"
go generate ./ent
go test ./...
go build -trimpath -ldflags='-s -w' -o /opt/xpanel/bin/xpanel-api ./cmd/api
go build -trimpath -ldflags='-s -w' -o /opt/xpanel/bin/xpanel-worker ./cmd/worker

cd web
corepack pnpm install --frozen-lockfile
corepack pnpm build
rm -rf /var/www/xpanel/*
cp -a dist/. /var/www/xpanel/
rm -rf node_modules

systemctl restart xpanel-api xpanel-worker
DOMAIN="$DOMAIN" bash "$SOURCE_DIR/scripts/configure-nginx.sh"
systemctl is-active --quiet xpanel-api xpanel-worker nginx
echo "XPanel v2 updated successfully."

