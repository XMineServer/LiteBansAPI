# LiteBans API

Read-only HTTP API over a LiteBans punishments database (bans, mutes, warnings, kicks) for the XMine platform, with public, player, and moderator endpoints.

## Overview

`litebans-api` exposes punishment data stored by the LiteBans plugin through a JSON HTTP API in the XMine service family. It serves three access tiers: public endpoints (a whitelisted subset of punishment types and aggregate stats), a player endpoint returning the authenticated caller's own punishments (JWT), and a moderator endpoint listing all punishments (gated by a permission). Permission checks for the moderator tier are delegated to the Authority service. The API is read-only; it never mutates the LiteBans database.

## Architecture

- `cmd/litebans-api/` — thin `main`: loads configuration and logging, then hands off to `internal/app` (`App.New` / `App.Run`).
- `internal/` — application packages:
  - `app` — wiring and lifecycle (construct dependencies, run the HTTP server, graceful shutdown).
  - `config` — environment-driven configuration with validation (source of truth for env vars).
  - `httpapi` — HTTP server, router, and OpenAPI strict handlers.
  - `service` / `domain` / `repository` / `db` — punishment query logic, domain types, and read access to the LiteBans MySQL database.
  - `auth` — JWT validation and Authority-backed permission checks.
  - `middleware` / `metrics` / `logging` / `requestid` — observability middleware, metrics, structured logging, and request-id propagation.
- `api/` — OpenAPI contract: `openapi.yaml` plus generated `api.gen.go` produced by [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) (`go generate ./api/...`).
- `clients/authority/` — generated client for the Authority service used for permission checks (`go generate ./clients/...`).

## Configuration

Configuration is read from environment variables (see `internal/config/config.go` and `.env.example`). Required variables must be set or the service exits on startup.

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `BUILD_TARGET` | Yes | — | Public host/IP identifier this instance serves. |
| `DATABASE_DRIVER` | Yes | — | Database driver; only `mysql` is supported. |
| `DATABASE_HOST` | Yes | — | LiteBans database host. |
| `DATABASE_PORT` | Yes | — | LiteBans database port. |
| `DATABASE_USER` | Yes | — | LiteBans database user. |
| `DATABASE_PASSWORD` | Yes | — | LiteBans database password. |
| `DATABASE_NAME` | Yes | — | LiteBans database name. |
| `TABLE_PREFIX` | No | `litebans_` | LiteBans table name prefix. |
| `CONSOLE_ALIASES` | No | `CONSOLE,Console` | Moderator names denoting the server console rather than a player. |
| `INCLUDE_INACTIVE` | No | `true` | Default visibility of inactive punishments in list endpoints. |
| `INCLUDE_SILENT` | No | `true` | Default visibility of silent punishments in list endpoints. |
| `DEFAULT_PAGE_SIZE` | No | `10` | Default page size (must be positive). |
| `MAX_PAGE_SIZE` | No | `100` | Maximum page size (must be ≥ `DEFAULT_PAGE_SIZE`). |
| `OBFUSCATE_IDS` | No | `false` | Obfuscate punishment ids in responses. |
| `OBFUSCATION_SECRET` | No | — | Secret for id obfuscation; required when `OBFUSCATE_IDS=true`. |
| `JWT_PUBLIC_KEY_PATH` | Yes | — | Path to the PEM public key issued by Identity (validates JWTs). |
| `JWT_ISSUER` | No | `xmine-identity` | Expected JWT issuer. |
| `AUTHORITY_API_URL` | Yes | — | Base URL of the Authority service. |
| `AUTHORITY_CACHE_TTL` | No | `60s` | TTL for cached Authority permission results. |
| `INTERNAL_TOKEN` | Yes | — | Shared secret authenticating internal calls to Authority. |
| `MOD_PERMISSION` | No | `web.litebans.view.all` | Permission required for the moderator endpoints. |
| `HTTP_ADDR` | No | `:8080` | HTTP listen address. |
| `LOG_LEVEL` | No | `info` | slog level (`debug`, `info`, `warn`, `error`). |
| `LOG_FORMAT` | No | `text` | Log output format (`json` or `text`). |

## Running locally

Prerequisites: Go 1.26.1, a reachable LiteBans MySQL database, and a reachable Authority service.

```sh
cp .env.example .env
# edit .env and fill in the required values (DATABASE_*, JWT_PUBLIC_KEY_PATH, AUTHORITY_API_URL, INTERNAL_TOKEN, ...)

go generate ./api/... ./clients/...
go run ./cmd/litebans-api
```

## Docker

```sh
docker build -t litebans-api .

docker run --rm -p 8080:8080 --env-file .env litebans-api
```

The container exposes port `8080`.

## API

The OpenAPI specification lives at `api/openapi.yaml` and is the source of truth for the HTTP contract. It is published to the [XMineDocs](https://github.com/XMineServer/XMineDocs) repository by the `publish-spec` GitHub workflow. Endpoint groups:

- **Public** (`/api/v1/public/*`) — active bans only (no other type is exposed publicly), aggregate punishment stats, and identity lookup by name or UUID (single or batch). Every `Punishment` already includes a resolved `playerName`; the batch lookup endpoint is for resolving other uuids client-side (e.g. moderator uuids) without N+1 calls.
- **Player** (`/api/v1/player/*`, JWT) — the caller's own punishments, across all types or filtered to one type.
- **Moderator** (`/api/v1/mod/*`, moderator permission) — full punishments listing across all players/moderators, across all types or filtered to one type.
- **Detail** (`/api/v1/{public,player,mod}/punishments/{type}/{id}`) — single-punishment lookup by type and id; one endpoint per access tier, each with its own visibility rule (public: active bans only; player: own records of any type plus any active ban; moderator: unrestricted).

Refer to the spec for request/response schemas and authentication requirements.

## Observability

- `GET /health` — health/liveness check.
- `GET /metrics` — Prometheus metrics (exposed via `promhttp`).
- Structured logging via `slog`: format is selected by `LOG_FORMAT` (`json` or `text`) and level by `LOG_LEVEL`. Records at `Error` level and above go to stderr; everything else goes to stdout.
