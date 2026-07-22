#!/usr/bin/env bash
set -Eeuo pipefail

DOMAIN="${DOMAIN:-x.unsvalo.com}"
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ ! -s "/etc/letsencrypt/live/${DOMAIN}/fullchain.pem" ]] || [[ ! -s "/etc/letsencrypt/live/${DOMAIN}/privkey.pem" ]]; then
  echo "TLS certificate for ${DOMAIN} was not found." >&2
  exit 1
fi

sed "s/__DOMAIN__/${DOMAIN}/g" "$SOURCE_DIR/deploy/nginx-cloudflare.conf.template" > /etc/nginx/sites-available/xpanel
ln -sfn /etc/nginx/sites-available/xpanel /etc/nginx/sites-enabled/xpanel
nginx -t
systemctl reload nginx
echo "Nginx configured for direct HTTPS and Cloudflare Flexible/Full modes."

