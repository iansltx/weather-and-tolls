# syntax=docker/dockerfile:1

ARG FRANKENPHP_VERSION=1.12.7
ARG PHP_VERSION=8.5

# --- Frontend ----------------------------------------------------------------
FROM node:26-alpine AS frontend
WORKDIR /src

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund

COPY frontend/ ./

# VERSION is baked into the bundle (used to detect new deployments) and into
# version.txt, which the poller watches to push version events.
ARG VERSION=dev
ENV VITE_APP_VERSION=$VERSION
RUN mkdir -p public && echo "$VERSION" > public/version.txt && npm run build

# --- PHP dependencies (Slim 4) -----------------------------------------------
FROM composer:2 AS php-deps
WORKDIR /app

COPY php/composer.json php/composer.lock ./
RUN composer install --no-dev --no-interaction --no-progress

# --- Custom FrankenPHP binary with the kiosk Go extension --------------------
# The kiosk extension (a PHP extension written in Go, loaded through
# FrankenPHP's bridge) is compiled into the FrankenPHP binary with xcaddy,
# per https://frankenphp.dev/docs/docker/#how-to-install-more-caddy-modules
FROM dunglas/frankenphp:${FRANKENPHP_VERSION}-builder-php${PHP_VERSION}-trixie AS builder
COPY --from=caddy:builder /usr/bin/xcaddy /usr/bin/xcaddy

COPY . /go/src/kiosk/

RUN CGO_ENABLED=1 \
    XCADDY_SETCAP=1 \
    XCADDY_GO_BUILD_FLAGS="-ldflags='-w -s' -tags=nobadger,nomysql,nopgx" \
    CGO_CFLAGS=$(php-config --includes) \
    CGO_LDFLAGS="$(php-config --ldflags) $(php-config --libs)" \
    xcaddy build \
        --output /usr/local/bin/frankenphp \
        --with github.com/dunglas/frankenphp=./ \
        --with github.com/dunglas/frankenphp/caddy=./caddy/ \
        --with github.com/dunglas/caddy-cbrotli \
        --with github.com/dunglas/mercure/caddy \
        --with github.com/dunglas/vulcain/caddy \
        --with ian.im/weather-and-tolls=/go/src/kiosk/

# --- Runtime -----------------------------------------------------------------
FROM dunglas/frankenphp:${FRANKENPHP_VERSION}-php${PHP_VERSION}-trixie AS runner

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/bin/frankenphp /usr/local/bin/frankenphp
COPY caddy/Caddyfile /etc/frankenphp/Caddyfile
COPY docker/entrypoint.sh /kiosk-entrypoint.sh
RUN chmod +x /kiosk-entrypoint.sh

# PHP app: front controller in the web root, vendor + sources + poller
# outside of it.
COPY php/index.php /app/public/index.php
COPY php/src/ /app/php/src/
COPY php/bin/ /app/php/bin/
COPY --from=php-deps /app/vendor/ /app/php/vendor/

# Built frontend: hashed assets + Vite manifest + version marker (no
# index.html; the PHP front controller renders the shell from the manifest).
COPY --from=frontend /src/dist/assets/ /app/public/assets/
COPY --from=frontend /src/dist/.vite/ /app/public/.vite/
COPY --from=frontend /src/dist/version.txt /app/public/version.txt

ENV SERVER_NAME=:8080 \
    KIOSK_WEBROOT=/app/public \
    MERCURE_INTERNAL_URL=http://127.0.0.1:8080/.well-known/mercure

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8080/api/version || exit 1

CMD ["/kiosk-entrypoint.sh"]
