#!/bin/sh
set -e

# If not running as root, just start the app directly
# (user already set via docker-compose user: directive)
if [ "$(id -u)" != "0" ]; then
    exec /app/holyhome "$@"
fi

# Running as root - handle permissions and drop privileges
PUID=${PUID:-$(id -u app)}
PGID=${PGID:-$(id -g app)}

# Update app user/group to match requested UID/GID
if [ "$PUID" != "$(id -u app)" ] || [ "$PGID" != "$(id -g app)" ]; then
    deluser app 2>/dev/null || true
    delgroup app 2>/dev/null || true
    addgroup -g "$PGID" -S app
    adduser -u "$PUID" -S app -G app
fi

# Fix ownership of data directory
chown -R app:app /data

# Drop privileges and run as app user
exec su-exec app /app/holyhome "$@"
