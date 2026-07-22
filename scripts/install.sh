#!/usr/bin/env bash
set -Eeuo pipefail

DOMAIN="${DOMAIN:-x.unsvalo.com}"
APP_ROOT="/opt/xpanel"
APP_USER="xpanel"
GO_VERSION="1.25.7"
NODE_VERSION="22.14.0"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Run this installer as root." >&2
  exit 1
fi

if [[ ! -f /etc/os-release ]]; then
  echo "Unsupported Linux distribution." >&2
  exit 1
fi
. /etc/os-release
case "${ID:-}" in ubuntu|debian) ;; *) echo "Supported systems: Ubuntu and Debian." >&2; exit 1 ;; esac

export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends ca-certificates curl git jq nginx openssl postgresql postgresql-client redis-server sudo xz-utils certbot python3-certbot-nginx

if ! id "$APP_USER" >/dev/null 2>&1; then
  useradd --system --home /var/lib/xpanel --create-home --shell /usr/sbin/nologin "$APP_USER"
fi
install -d -m 0755 "$APP_ROOT/bin" /var/www/xpanel
install -d -o "$APP_USER" -g "$APP_USER" -m 0750 /var/lib/xpanel /etc/xpanel

if ! swapon --show --noheadings | grep -q . && [[ ! -f /swapfile ]]; then
  fallocate -l 1G /swapfile || dd if=/dev/zero of=/swapfile bs=1M count=1024
  chmod 600 /swapfile
  mkswap /swapfile >/dev/null
  swapon /swapfile
  echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

install_go() {
  local archive="go${GO_VERSION}.linux-amd64.tar.gz"
  local tmp="/tmp/${archive}"
  local expected
  expected="$(curl -fsSL 'https://go.dev/dl/?mode=json&include=all' | jq -r --arg file "$archive" '.[] | .files[] | select(.filename == $file) | .sha256' | head -n1)"
  [[ -n "$expected" && "$expected" != "null" ]] || { echo "Unable to obtain Go checksum." >&2; exit 1; }
  curl -fsSL "https://go.dev/dl/${archive}" -o "$tmp"
  echo "${expected}  ${tmp}" | sha256sum -c -
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "$tmp"
  rm -f "$tmp"
}

if [[ ! -x /usr/local/go/bin/go ]] || [[ "$(/usr/local/go/bin/go version 2>/dev/null || true)" != *"go${GO_VERSION}"* ]]; then
  install_go
fi
export PATH="/usr/local/go/bin:/usr/local/node/bin:${PATH}"

install_node() {
  local archive="node-v${NODE_VERSION}-linux-x64.tar.xz"
  local tmp="/tmp/${archive}"
  curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/${archive}" -o "$tmp"
  curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/SHASUMS256.txt" -o /tmp/node-shasums.txt
  grep " ${archive}$" /tmp/node-shasums.txt | sed "s# ${archive}\$# ${tmp}#" | sha256sum -c -
  rm -rf /usr/local/node
  mkdir -p /usr/local/node
  tar -C /usr/local/node --strip-components=1 -xJf "$tmp"
  rm -f "$tmp" /tmp/node-shasums.txt
}

if [[ ! -x /usr/local/node/bin/node ]] || [[ "$(/usr/local/node/bin/node --version 2>/dev/null || true)" != "v${NODE_VERSION}" ]]; then
  install_node
fi
corepack enable
corepack prepare pnpm@11.9.0 --activate

DB_PASSWORD="$(openssl rand -hex 24)"
REDIS_PASSWORD="$(openssl rand -hex 24)"
MASTER_KEY="$(openssl rand -base64 48 | tr -d '\n')"
ADMIN_USERNAME="admin"
ADMIN_PASSWORD="$(openssl rand -base64 24 | tr -d '\n' | tr '/+' 'Xy')"

systemctl enable --now postgresql redis-server
if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='xpanel'" | grep -q 1; then
  sudo -u postgres psql -v ON_ERROR_STOP=1 -c "CREATE ROLE xpanel LOGIN PASSWORD '${DB_PASSWORD}'"
  sudo -u postgres createdb -O xpanel xpanel
else
  sudo -u postgres psql -v ON_ERROR_STOP=1 -c "ALTER ROLE xpanel WITH PASSWORD '${DB_PASSWORD}'"
fi

sed -ri 's/^#?bind .*/bind 127.0.0.1 -::1/' /etc/redis/redis.conf
sed -ri '/^requirepass /d;/^maxmemory /d;/^maxmemory-policy /d' /etc/redis/redis.conf
cat >> /etc/redis/redis.conf <<EOF
requirepass ${REDIS_PASSWORD}
maxmemory 128mb
maxmemory-policy allkeys-lru
EOF
systemctl restart redis-server

cat > /etc/xpanel/xpanel.env <<EOF
APP_ADDRESS=127.0.0.1:8090
DATABASE_URL=postgres://xpanel:${DB_PASSWORD}@127.0.0.1:5432/xpanel?sslmode=disable
REDIS_ADDRESS=127.0.0.1:6379
REDIS_PASSWORD=${REDIS_PASSWORD}
REDIS_DB=0
MASTER_KEY=${MASTER_KEY}
COOKIE_SECURE=true
SESSION_TTL=24h
EOF
chown root:"$APP_USER" /etc/xpanel/xpanel.env
chmod 0640 /etc/xpanel/xpanel.env

cd "$SOURCE_DIR"
/usr/local/go/bin/go generate ./ent
/usr/local/go/bin/go build -trimpath -ldflags='-s -w' -o "$APP_ROOT/bin/xpanel-api" ./cmd/api
/usr/local/go/bin/go build -trimpath -ldflags='-s -w' -o "$APP_ROOT/bin/xpanel-worker" ./cmd/worker
chown -R root:root "$APP_ROOT"
chmod 0755 "$APP_ROOT/bin/xpanel-api" "$APP_ROOT/bin/xpanel-worker"

cd "$SOURCE_DIR/web"
/usr/local/node/bin/corepack pnpm install --frozen-lockfile
/usr/local/node/bin/corepack pnpm build
rm -rf /var/www/xpanel/*
cp -a dist/. /var/www/xpanel/
rm -rf node_modules

install -m 0644 "$SOURCE_DIR/deploy/xpanel-api.service" /etc/systemd/system/xpanel-api.service
install -m 0644 "$SOURCE_DIR/deploy/xpanel-worker.service" /etc/systemd/system/xpanel-worker.service
sed "s/__DOMAIN__/${DOMAIN}/g" "$SOURCE_DIR/deploy/nginx.conf.template" > /etc/nginx/sites-available/xpanel
ln -sfn /etc/nginx/sites-available/xpanel /etc/nginx/sites-enabled/xpanel
rm -f /etc/nginx/sites-enabled/default
nginx -t
systemctl daemon-reload
systemctl enable --now xpanel-api xpanel-worker nginx

set -a
. /etc/xpanel/xpanel.env
set +a
ADMIN_USERNAME="$ADMIN_USERNAME" ADMIN_PASSWORD="$ADMIN_PASSWORD" "$APP_ROOT/bin/xpanel-api" bootstrap-admin

if getent ahostsv4 "$DOMAIN" >/dev/null 2>&1; then
  certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos --register-unsafely-without-email --redirect || true
fi

cat > /root/xpanel-v2-credentials.txt <<EOF
URL: https://${DOMAIN}
Username: ${ADMIN_USERNAME}
Password: ${ADMIN_PASSWORD}
EOF
chmod 0600 /root/xpanel-v2-credentials.txt

/usr/local/go/bin/go clean -modcache || true
apt-get clean
rm -rf /var/lib/apt/lists/*

echo "XPanel v2 installation completed."
echo "Credentials: /root/xpanel-v2-credentials.txt"
