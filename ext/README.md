# kiosk extension

A PHP extension written in Go, loaded through FrankenPHP's Go/PHP bridge.
Exposes the kiosk's data-source lookups and Mercure interactions as native
PHP functions; the Slim 4 routes and the poller call these.

## API

- `kiosk_resolve_coords(string $location): array` — `{"lat": float, "lon": float}`
- `kiosk_fetch_weather(float $lat, float $lon): array` — `{"location": string, "temperature": float, "humidity": int, "dailyHigh": float, "dailyLow": float, "units": string, "observedAt": string}`
- `kiosk_fetch_tolls(): array` — `{"requestedAt": string, "rates": list<{"name": string, "tollTagRate": float, "payByMailRate": float, "closed": bool}>}`
- `kiosk_mercure_subscriptions(): array` — `{"topics": list<string>}` (active subscription topics)
- `kiosk_mercure_publish(string $topic, string $data, string $type, string $id): array` — `{"id": string}` (hub-assigned update ID)

Errors from the Go layer throw a `RuntimeException` instead of returning an
error payload.

## Maintenance

The initial boilerplate (this module's `kiosk.c`, arginfo, stub) was
generated with `frankenphp extension-init`. It is now **hand-maintained**:
`kiosk.c` propagates Go errors as PHP exceptions, which the generator cannot
express — re-running `frankenphp extension-init` would drop that behavior, so
edit the files by hand instead. `kiosk.go` documents the error contract.
