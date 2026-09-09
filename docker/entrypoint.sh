#!/bin/sh
# Runs the FrankenPHP server and the PHP poller side by side. Both use the
# same custom binary, so the Kiosk\Bridge Go extension is available to the
# routes (Slim 4) and to the poller's PHP loop.
frankenphp run --config /etc/frankenphp/Caddyfile --adapter caddyfile &
SERVER_PID=$!

frankenphp php-cli /app/php/bin/poller.php &
POLLER_PID=$!

shutdown() {
	kill "$SERVER_PID" "$POLLER_PID" 2>/dev/null || true
	exit 0
}

trap shutdown TERM INT
wait "$SERVER_PID" "$POLLER_PID"
shutdown