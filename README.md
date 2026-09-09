# Weather and Tolls

A kiosk-style web display for Austin-area weather and toll lane prices,
built on PHP/FrankenPHP to demo its Go/PHP extensibility:

- Weather data for a configurable location, via OpenWeather
  - Current temperature
  - Humidity
  - Expected daily high and low
  - Last refresh time and countdown to the next refresh
- Austin-area variable-rate toll lane prices
  - Toll tag rates by default
  - Pay-by-mail rates via `Ctrl-T`

```
Browser (Vue) ──SSE──> Mercure hub ──<── PHP poller ──> kiosk_* Go extension ──> OpenWeather / Mobility Authority
      │                                                          (FrankenPHP Go/PHP bridge)
      └────HTTP────> Slim 4 (PHP worker) ────────┘
```

## How it works

- **Routes** are plain PHP on [Slim 4](https://www.slimframework.com/)
  (`php/`): `/api/state`, `/api/weather`, `/api/tolls`, `POST /api/refresh`,
  `/api/version`, `/api/health`, plus the app shell, which is rendered from
  the Vite build manifest. The front controller runs in FrankenPHP worker
  mode (`frankenphp_handle_request()`): the Slim app is booted once and kept
  resident instead of being re-executed for every request.
- **Data-source calls** are methods on a native PHP class implemented in Go
  (`ext/`), loaded as a PHP extension inside a custom FrankenPHP binary:
  `Kiosk\Bridge::resolveCoords()`, `::fetchWeather()`, `::fetchTolls()`,
  `::mercureSubscriptions()`, `::mercurePublish()`. The methods wrap the
  clients in `apiclients/`, and API keys stay inside the Go layer, never
  reaching the client.
- **The poller** (`php/bin/poller.php`) is a PHP loop that runs alongside the
  server. It polls upstream only while clients are connected (detected via
  `Kiosk\Bridge::mercureSubscriptions()`), keeps the same refresh interval /
  shared-cadence debounce / exponential backoff behavior as the original
  terminal UI, and publishes the snapshot over Mercure after every completed
  refresh cycle — so "as of" times and refresh countdowns stay live, and any
  data change reaches the browser within seconds of the poll. It also watches
  `version.txt` in the web root and pushes a version event when a new
  frontend is deployed.
- **The frontend** (`frontend/`, Vue 3 + Vite) fetches one state snapshot on
  load, then subscribes to the Mercure hub — no client-side polling.
  Keyboard shortcuts: `Ctrl-R` triggers a server-side refresh, `Ctrl-T`
  toggles toll tag/pay-by-mail rates. When a new frontend version is
  deployed, open pages show "New version available, click to refresh";
  clicking reloads the page.
- The former CLI parameters map to query string parameters: `location` (and
  `refresh`, in seconds) on `/api/state` and `/api/refresh`. The location can
  also be passed as a page query parameter (`/?location=78757`).

## Running with Docker

Requirements: Docker, an OpenWeather API key.

```sh
cp .env.example .env       # then set OPENWEATHER_API_KEY (and Mercure keys)
make web-up                # builds the image and starts the server on :8080
```

`make web-up` runs `docker compose up -d`; logs via `make web-logs`, stop via
`make web-down`. The image is a custom FrankenPHP build: the Go extension is
compiled into the binary with xcaddy, the frontend is built in an earlier
stage, and the container runs the server and the PHP poller side by side.

The app is then available at `http://localhost:8080`.

### Configuration (environment variables)

| Variable | Default | Purpose |
| --- | --- | --- |
| `OPENWEATHER_API_KEY` | — | OpenWeather API key (required) |
| `DEFAULT_WEATHER_LOCATION` | `Austin, TX, US` | Location when the page does not pass `?location=` |
| `KIOSK_REFRESH_SECONDS` | `300` | Automatic refresh interval in seconds |
| `MERCURE_PUBLISHER_JWT_KEY` | — | Mercure publisher JWT signing key |
| `MERCURE_SUBSCRIBER_JWT_KEY` | — | Mercure subscriber JWT signing key (subscription listing API) |
| `MERCURE_INTERNAL_URL` | `http://127.0.0.1:80/.well-known/mercure` | Hub endpoint used by the Go extension |
| `OPENWEATHER_BASE_URL` | `https://api.openweathermap.org` | Override for API testing |
| `TOLL_RATES_URL` | `https://www.mobilityauthority.com/wp-admin/admin-ajax.php` | Override for API testing |
| `KIOSK_SCAN_SECONDS` | `5` | Subscription scan interval |
| `KIOSK_VERSION_SECONDS` | `30` | Frontend version watch interval |

Mercure keys are set through environment variables in `compose.yaml` (with
dev defaults); the browser subscribes anonymously.

## HTTP API

- `GET /api/state[?location=L][&refresh=S]` — full snapshot (weather, tolls,
  errors, next refresh times) plus the Mercure topic to subscribe to.
- `GET /api/weather?location=L` — live weather lookup.
- `GET /api/tolls` — live toll rate lookup.
- `POST /api/refresh?location=L` — asks the poller for an immediate refresh;
  the result is pushed over Mercure (bound to `Ctrl-R`).
- `GET /api/version`, `GET /api/health`.
- `GET /.well-known/mercure` — Mercure hub (SSE).

## Tests

```sh
make test    # API client tests (mocked upstreams; no network required)
```
