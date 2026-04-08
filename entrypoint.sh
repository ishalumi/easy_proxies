#!/bin/sh
# Fix ownership of mounted config directory so the non-root user can write
chown -R easy:easy /etc/easy_proxies 2>/dev/null || true
chown -R easy:easy /app 2>/dev/null || true

exec gosu easy /usr/local/bin/easy_proxies "$@"
