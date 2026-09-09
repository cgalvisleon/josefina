# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Run server
```bash
gofmt -w . && go run ./cmd/server -port 1377 -rpct 4377 -name josefina
```
Flags (see `cmd/server/main.go`): `-port` (HTTP, env `PORT`, default 1377), `-rpct` (RPC port, env `RPC_PORT`, default 4370), `-name` (env `DB_NAME`, default `josefina` — currently unused past this point, no code reads it back), `-path_data`/`-path_wald`/`-path_system` (env `DB_PATH_DATA`/`DB_PATH_WAL`/`DB_PATH_SYSTEM`).

There is no `cmd/client` and no REPL — Josefina is driven entirely over its HTTP API (`pkg/server`); see Architecture below.

### Test
```bash
go test ./...
```
There are currently **no `_test.go` files anywhere in this repo** — this is not a "run the suite" codebase yet. If you add tests, `go test ./path/to/pkg/...` and `go test -run TestName ./path/to/pkg/` work as usual.

### Build
```bash
go build ./cmd/server
```
`cmd/store` and `cmd/test` are standalone scratch programs exercising `internal/store` and `internal/jdb` directly (not part of the product surface, but still `go build`-able).

### Format
```bash
gofmt -w .
```

### Go version
```bash
goenv local 1.25.0   # project uses Go 1.25.0 (see go.mod, .go-version)
```

This repo also participates in the sibling `go.work` at the `cgalvisleon/` workspace root (alongside `et`, `core-studio/api`, `tick`), so local edits to `github.com/cgalvisleon/et` are picked up live without a published release — run `go env GOWORK` to confirm it's active.

## Code style

### Comments
All doc comments for functions, methods, and types must use this block style:

```go
/**
* FunctionName: Brief description.
* @param paramName type
* @return type
**/
```

- Use `@param` for each parameter and `@return` for the return value(s).
- Inline comments inside function bodies stay as `//`.
- Never use single-line `//` doc comments above a function or type declaration.

## Architecture

Josefina is a custom document database engine written in Go, exposed only over an **HTTP + JSON API** (module `github.com/josefina`). There is no SQL text parser, no TCP client protocol, and no REPL in the current codebase — a prior iteration had those (see "Stale docs" below), but that layer has been removed. Queries and commands are JSON documents dispatched to a fluent Go query/command builder underneath.

### Entry point
- `cmd/server/main.go` — the only product entry point. Wires env vars/flags, calls `internal/server.New()`, then `.Start()`.
- `cmd/store/main.go`, `cmd/test/main.go` — scratch programs exercising `internal/store` and `internal/jdb.Load()` directly.

### Request flow
```
cmd/server → internal/server.New() → internal/server/v1.New()
    → jdb.Load()                         (boots the singleton *jdb.Server)
    → pkg/server.Routes(name, version, server)  (mounts HTTP routes on it)
    → github.com/cgalvisleon/et/server.Ettp     (actual HTTP listener, mounted at "/" and "/v1")
```
`pkg/server` (package `server`, not to be confused with `internal/server`) defines the REST surface using `github.com/cgalvisleon/et/router`:
- `POST /signin`, `POST /signout` — issue/revoke a JWT (`github.com/cgalvisleon/et/claim`) tied to a `jdb.Session`
- `POST /system` — batches `define`/`describe` JSON ops (create/inspect database, schema, model, user)
- `POST /query` — batches `query` JSON ops (currently a no-op stub, `execQuery` in `internal/jdb/query.go`)
- `POST /command` — batches `insert`/`update`/`delete`/`bulk` JSON ops
- `POST /uploadXls`, `POST /uploadCsv`, `POST /uploadDb` — multipart/JSON bulk-load endpoints (XLS, CSV, or a live import from Postgres/MySQL/SQLite/Oracle/SQL Server via `internal/jdb/datasource.go`)

Every authenticated route reads a `Bearer` token, validated by `pkg/server.Authentication` (checks the JWT via `claim.ParceToken`, then looks up the session via `jdb.Authenticate`) before the handler runs. `POST /system`, `/query`, `/command`, and the uploads are wired as `Public` routes in the router but still require and check the bearer token by hand inside `internal/jdb/server.go` (`jSystem`/`jQuery`/`jCommand`/`jUpload*`) — don't assume the router's own `Authentication` middleware is what's gating them.

Inside `internal/jdb`, each of `jSystem`/`jQuery`/`jCommand` fans its JSON array fields out into `queryJob`s (`internal/jdb/query.go`) and runs them concurrently via `runQueryJobs`, merging results into one `et.Items`. All top-level DB work — including simple reads — is additionally serialized through a worker-pool queue: `Server.Exec(params, fn)` pushes a `Request` onto a buffered channel and blocks on its `Response`; `Server.runWorkers()` starts `cores*2+1` (or `DB_POOL`) goroutines consuming that queue. Any new server-level operation should go through `Server.Exec`, not call its handler directly.

### Layer breakdown

**`internal/store`** — Low-level WAL file storage engine, unchanged in spirit from earlier versions
- `FileStore`: append-only segmented file store (`segment-XXXXXX.dat`) with an in-memory index (`map[string]*RecordRef`)
- Tombstone-based deletes; automatic compaction (`compact.go`) once tombstones exceed a threshold; snapshots on segment roll-over (`snapshot.go`)
- `wal.go` adds `WalEntry`/`ApplyWalEntry`/`WalSince` — a replication log primitive not yet wired to any clustering code (see `config.json` note below)

**`internal/jdb`** — Server, schema, catalog, and query/command engine (the bulk of the codebase, ~8k lines)
- Hierarchy: package-level singleton `*Server` (`jdb.go`/`server.go`) → `DB` (`db.go`) → `Schema` (`schema.go`) → `Model` (`model.go`) → `Field`/`Index`/`Detail` (`field.go`, `define.go`)
- `Server` also owns cross-database concerns: `Users`/`Session`s (`users.go`, `session.go` — JWT-backed, admin bootstrap via `USER_ADMIN`/`PASSWORD_ADMIN` env, default `admin`/`admin`), `Series` (`series.go`, named auto-increment/format counters), `Cache` (`cache.go`), and `Instances` (`instances.go` — async job/audit tracking used by the upload endpoints)
- `Model` owns a primary `FileStore` (key → document) plus one `FileStore` + one in-memory `BTree` (`btree.go`, B+ tree, degree 32) per secondary index; each `BTree` self-persists and rebuilds from its `FileStore` on `Model.Init()`
- Query DSL: `Model.Where(cond)` → `*Query`, `Model.Insert/Update/Delete/Upsert/Bulk(...)` → `*Command` (`cmd.go`, `where.go`), both fluent (`.And`, `.Or`, `.Selects`, `.Hidden`, `.OrderBy`, `.Limit`, `.Exec()`/`.One()`). Conditions are built with helpers in `condition.go` (`Eq`, `Neg`, `Less`, `More`, `Like`, `In`, `Between`, `Null`, ...) which produce `*et.Condition`, matched against `BTree` indexes using typed `IndexKey`s (`key.go` — numeric vs. lexicographic ordering is type-aware, `KeyFromAny` auto-detects from decoded JSON)
- JavaScript triggers (`Model.AddBeforeInsert`/`AddAfterInsert`/... and `fireTriggers` in `model.go`) run through `github.com/cgalvisleon/et/jrex` (`jrex.NewInstance()`)
- `datasource.go` builds DSNs and imports an external table (Postgres/MySQL/SQLite/Oracle/SQL Server via `lib/pq`, `go-sql-driver/mysql`, `modernc.org/sqlite`, `sijms/go-ora`, `microsoft/go-mssqldb`) row-by-row into a `Model`

**`internal/server` / `internal/server/v1`** — Thin wiring layer: builds the `et/server.Ettp` HTTP server, loads `jdb`, mounts `pkg/server`'s routes at `/` and `/v1`, registers the close hook (`jrpc.Close()`)

**`pkg/server`** — The public REST API: routing (`router.go`), bearer-token auth middleware (`authentication.go`), event subscription wiring (`events.go`) via `github.com/cgalvisleon/et/event`

**`internal/msg`** — Centralized error/message string constants (`MSG_*`), with English/Spanish variants selected by the `LANG` env var

### Configuration
- `config.json` — a `nodes` list of `host:port` cluster peer addresses; still not read by any Go code. Combined with the unused `internal/store/wal.go` replication primitives, this documents an intended-but-unbuilt clustering feature — don't assume nodes coordinate with each other.
- `.env` — read via `envar`; besides the server flags above, includes `RELSEG_SIZE`, `SYNC_ON_WRITE`, `TIMEZONE`, `MIN_THRESHOLD_COMPACT`, `TTL_TRANSACCION`, `DB_POOL`, `PATH_URL`, plus `REDIS_*`/`NATS_*` used by `et`'s `cache`/`event` packages
- Data is persisted under `DB_PATH_DATA` / `DB_PATH_WAL` / `DB_PATH_SYSTEM` (defaults under `./data/`)

### Key dependency
- `github.com/cgalvisleon/et` — shared utilities: `server` (HTTP host, `Ettp`), `router` (chi-based REST routing), `response` (JSON response helpers), `claim` (JWT), `event` (pub/sub), `cache`, `jrex` (JS trigger runtime), `csv`/`xls` (bulk import parsing), `logs`, `envar`, `utility`, `reg`

### Stale docs, don't rely on them
- `README.md` and `AGENTS.md` describe an even older architecture than either of those files' own history suggests: a `cmd/client` REPL, TCP remote/embedded modes, SQL-text syntax (`CREATE TABLE`, `SELECT ... WHERE`, `SET SQL STATE`), and an `internal/catalog` package. **None of that exists in the current tree** — `cmd/client`, `internal/catalog`, `internal/stmt`, `internal/jsql`, `internal/cli`, `internal/client`, `pkg/http`, and `pkg/websocket` are all gone. The current query/command surface is the JSON-based `internal/jdb` DSL described above, driven only over HTTP. Trust this file and the source over those two.
