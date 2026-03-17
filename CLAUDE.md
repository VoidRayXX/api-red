# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run the server (port 8080)
go run main.go

# Build binary
go build -o api-red .

# Verify compilation without running
go build ./...

# Download dependencies
go mod download
```

There are no tests in this codebase.

## Architecture

This is a Go REST API that scrapes and proxies public transit data for Santiago, Chile. It exposes 4 endpoints by aggregating data from external websites and APIs.

### Parser pattern

`main.go` defines a `Parser` interface. Each module implements it:

```go
type Parser interface {
    GetRoute() string       // e.g. "bus-stop/:stopid"
    StartParser()           // initialize HTTP clients, sessions, regex
    Parse(c *gin.Context)   // gin request handler
    StopParser()
    GetCronTasks() []*common.CronTask
}
```

At startup, `main.go` loops over all parsers, calls `StartParser()`, registers the route, and schedules any cron tasks (also executing them immediately on startup).

### Modules

| Package | Endpoint | Data source |
|---|---|---|
| `balance/` | `GET /balance/:bipid` | Scrapes `cargatubip.metro.cl` — maintains a JSF ViewState token |
| `busstop/` | `GET /bus-stop/:stopid` | Scrapes `web.smsbus.cl` — maintains a JSESSIONID cookie, retries once on session expiry |
| `metronetwork/` | `GET /metro-network` | Scrapes `metro.cl/tu-viaje/estado-red` — caches station schedules and holiday status in memory |
| `bus/` | `GET /bus/:stopid` | Proxies `red.cl` predictor API — extracts JWT from page HTML on each request |

### metronetwork cron tasks

This is the most complex module. On startup and then on a schedule it:
- **Midnight daily**: fetches Chilean holidays for the current year from `apis.digital.gob.cl`
- **1am daily**: fetches schedules for all ~400 metro stations from `metro.cl/api/horariosEstacion.php`

Station open/closed status is computed at request time in `CompositeTime.IsClosed()` using the cached schedules and holiday flag.

### Session management (busstop)

`busstop.Parser` holds a single `*http.Client` (10s timeout) and a `JSESSIONID` session cookie obtained at startup. On each request it uses the cached session. If the response comes back with an empty stop name (indicating session expiry), it refreshes the session and retries the fetch once.

### Error codes

Each module defines its own `ErrorCode` map with Spanish descriptions. Code `0` always means success. HTTP status codes: `400` for client/parsing errors, `500` for upstream failures.
