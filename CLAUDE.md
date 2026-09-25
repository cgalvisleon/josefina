# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Run server
```bash
gofmt -w . && go run ./cmd/server -port 1377 -rpct 4377 -name josefina
```
Flags (see `cmd/server/main.go`): `-port` (HTTP, env `PORT`, default 1370 — `deploy.sh` and the command above use 1377), `-rpct` (RPC port, env `RPC_PORT`, default 4370), `-name` (env `DB_NAME`, default `josefina` — nothing reads it back), `-path_data` (env `DB_PATH_DATA`, default `./data/collections`), `-path_wald` (env `DB_PATH_WALD`), `-path_system` (env `DB_PATH_SYSTEM`).

Gotcha: `main.go` writes the WAL path to `DB_PATH_WALD`, but `internal/jdb/jdb.go` reads `DB_PATH_WAL` — so `-path_wald` has no effect; set `DB_PATH_WAL` in `.env` instead.

There is no `cmd/client` and no REPL — Josefina is driven entirely over its HTTP API (`pkg/server`).

### Build, vet, format
```bash
go build ./...        # product binary: go build ./cmd/server
go vet ./...
gofmt -w .
```
`cmd/store` and `cmd/test` are standalone scratch programs exercising `internal/store` and `internal/jdb.Load()` directly — not part of the product surface.

### Test
There are **no `_test.go` files** in this repo yet. If you add tests: `go test ./internal/jdb/...` or `go test -run TestName ./internal/store/`.

### Go version and workspace
Go 1.25.0 (`go.mod`, `.go-version`; `goenv local 1.25.0`). The repo is part of the `go.work` at the `cgalvisleon/` workspace root, so edits to `github.com/cgalvisleon/et` are picked up live — `go env GOWORK` confirms it. When `et` changes its API, this repo must be updated in the same pass (recent examples: `et.And`/`et.Or` → `et.AND`/`et.OR`, `et.ToCondition` now returns `(*et.Condition, error)`, `claim.NewToken` → `claim.NewAuthenticationToken`, claims carry `UserID`/`TenantID`/`RoleID` instead of `SessionID`, `reg.GetUUID(id)` to default an empty id).

### Deploy / versioning
`version.sh` bumps the git tag (`--major`, `--minor`, `--request`). `deploy.sh` renders `deployments/template-deploy.yml` → `deployments/deploy.yml` for Kubernetes (image `cgalvisleon/josephine`, namespace `prod` on `main`, otherwise a dev profile); `deployments/local.yml` and `template-statefulset.yml` are the other manifests. The deployed service/image and default `PATH_URL` are `josephine`; the Go module and app name are `josefina`.

## Code style

All doc comments for functions, methods, and types use this block style (never a single-line `//` doc comment above a declaration; `//` stays for inline comments inside bodies):

```go
/**
* FunctionName: Brief description.
* @param paramName type
* @return type
**/
```

Other conventions visible across the code:
- Data travels as `et.Json` (map) / `et.Items` (`{Ok, Count, Result []et.Json}`), not typed structs, between layers. Domain structs expose `ToJson()`.
- User-facing error strings are `MSG_*` constants in `internal/msg` (English/Spanish selected by `LANG`); errors are built with `errors.New(msg.MSG_...)`, not inline literals. Sentinel errors like `ErrorSessionNotFound` are checked with `errors.Is`.
- Maps shared across goroutines are guarded by a `sync.RWMutex` next to them (e.g. `Errors`, the server's db map).

## Architecture

Josefina is a custom document database engine in Go (module `github.com/josefina`), exposed only over an **HTTP + JSON API**. There is no SQL text parser, TCP client protocol, or REPL — requests are JSON documents dispatched to a fluent Go query/command builder.

### Request flow
```
cmd/server/main.go → internal/server.New() → internal/server/v1.New()
    → jdb.Load()                               (boots the package-level singleton *jdb.Server)
    → pkg/server.Routes(name, version int, server)  (mounts routes under PATH_URL, default /josephine)
    → github.com/cgalvisleon/et/server.Ettp    (HTTP listener, mounted at "/" and "/v1")

HTTP handler (pkg/server/router.go)
    → GetBearerToken + response.GetBody, body.Set("token", token)
    → jdb.System / JQuery / JCommand / JUpload*   (public funcs in internal/jdb/jdb.go)
    → server.Exec(params, server.jX)             (worker-pool queue)
    → Server.jX: validate token → getSession → session.DB
    → DB.jX: split body into queryJobs → runQueryJobs (concurrent) → merged et.Items
    → response.ITEMS(w, r, 200, result)
```

### REST surface (`pkg/server`)
`pkg/server` (package `server`, not to be confused with `internal/server`) uses `github.com/cgalvisleon/et/router`:
- `GET /version`, `GET /routes` — public info
- `POST /signin`, `POST /signout` — issue/revoke a JWT (`et/claim`) tied to a `jdb.Session`
- `POST /system` — body keys `define` / `describe`, each an array of `{ "database" | "schema" | "model" | "user": {...} }`
- `POST /query` — body key `query` (currently a no-op stub: `execQuery` in `internal/jdb/query.go` returns empty)
- `POST /command` — body keys `insert` / `update` / `delete` / `bulk` (arrays; `update`/`delete` entries carry a `where` array parsed with `et.ToCondition`). `upsertQuery` exists but is not wired into `jCommand`.
- `POST /uploadXls`, `/uploadCsv`, `/uploadDb` — bulk loads (XLS, CSV, or a live import via `internal/jdb/datasource.go`)

**Auth:** all JDB routes are registered as `Public`, so the router's `Authentication` middleware (`pkg/server/authentication.go`, which puts claim fields into the request context) does **not** gate them. Each `Server.jSystem`/`jQuery`/`jCommand`/`jUpload*` in `internal/jdb/server.go` checks the bearer token by hand and resolves the session; the session's `DB` is the database the request runs against. Sessions are keyed by the claim's `UserID`.

### Layer breakdown

**`internal/store`** — Low-level storage engine
- `FileStore`: append-only segmented files (`segment-XXXXXX.dat`) with an in-memory index (`map[string]*RecordRef`)
- Tombstone deletes; automatic compaction (`compact.go`) past `MIN_THRESHOLD_COMPACT`; snapshots on segment roll-over (`snapshot.go`)
- `wal.go` (`WalEntry`/`ApplyWalEntry`/`WalSince`) — replication primitive not wired to anything yet

**`internal/jdb`** — Server, catalog, and query/command engine (~8k lines, the bulk of the code)
- Hierarchy: singleton `*Server` (`jdb.go`/`server.go`) → `DB` (`db.go`) → `Schema` (`schema.go`) → `Model` (`model.go`) → `Field`/`Index`/`Detail` (`field.go`, `define.go`). The server's own catalog lives in a `system` `FileStore`; on first boot it creates a default database.
- `Server` owns cross-database concerns: `Users`/`Session`s (`users.go`, `session.go` — admin bootstrap via `USER_ADMIN`/`PASSWORD_ADMIN`, default `admin`/`admin`), `Series` (`series.go`, named counters), `Cache` (`cache.go`), `Instances` (`instances.go`, async job/audit tracking for uploads). Each `DB` also keeps an internal errors model (`errors.go`).
- `Model` owns a primary `FileStore` (key → document) plus, per secondary index, a `FileStore` + in-memory B+ tree (`btree.go`, degree 32) that self-persists and rebuilds on `Model.Init()`.
- Query DSL: `Model.Where(cond)` → `*Query` (`where.go`); `Model.Insert/Update/Delete/Upsert/Bulk(...)` → `*Command` (`cmd.go`). Both fluent: `.And`, `.Or`, `.Selects`, `.Hidden`, `.OrderBy`, `.Limit`, `.Exec()`/`.One()`. The first condition keeps `et.NaC`; later ones default to `et.AND`. `Where` groups conditions by lower-cased field name.
- Conditions come from helpers in `condition.go` (`Eq`, `Neg`, `Less`, `More`, `Like`, `In`, `Between`, `Null`, ...) producing `*et.Condition`, evaluated by `BTree.ApplyConditions` (AND intersects, OR unions) over typed `IndexKey`s (`key.go` — numeric vs. lexicographic ordering; `KeyFromAny` infers from decoded JSON).
- JavaScript triggers (`Model.AddBeforeInsert`/`AddAfterInsert`/... → `fireTriggers`) run in `et/jrex`.
- `tx.go` only declares a `Transaction` record/status type (plus `DB.TransactionTTL` from `TTL_TRANSACCION`); there is no `Tx` type or commit/rollback flow yet, despite a `tx *Tx` mention in a `model.go` doc comment.
- `datasource.go` builds DSNs and imports an external table row by row (Postgres `lib/pq`, MySQL `go-sql-driver/mysql`, SQLite `modernc.org/sqlite`, Oracle `sijms/go-ora`, SQL Server `microsoft/go-mssqldb`).

**`internal/server` / `internal/server/v1`** — Wiring only: builds `et/server.Ettp`, loads `jdb`, mounts routes at `/` and `/v1`, registers the close hook (`jrpc.Close()`).

**`pkg/server`** — Public REST API: `router.go`, `authentication.go`, `events.go` (subscriptions via `et/event`).

**`internal/msg`** — `MSG_*` message constants, English/Spanish by `LANG`.

## Patterns for extending

- **New server-level operation:** add a lowercase `(s *Server) jX(params et.Json) (et.Items, error)` in `internal/jdb/server.go` that validates the token and resolves the session, then a public wrapper `JX(params)` in `jdb.go` that checks `server != nil` and calls `server.Exec(params, server.jX)`. Never call the handler directly — `Server.Exec` pushes onto the request channel consumed by `cores*2+1` workers (or `DB_POOL`).
- **New JSON op inside an endpoint:** write a `func xQuery(db *DB, params []et.Json) ([]et.Json, error)` in `query.go` and add `{xQuery, params.ArrayJson("x")}` to the `queryJob` list in the matching `DB.jSystem`/`jQuery`/`jCommand` (`db.go`). Jobs run concurrently; the first error wins.
- **New catalog object for `define`/`describe`:** add a `case` in `defineQuery`/`describeQuery` that resolves the target DB with `db.server.getDb(define.Str("database"))` and delegates to a `DB.defineX`/`describeX` method.
- **New HTTP route:** handler in `pkg/server/router.go` follows the existing shape (bearer token → body → `body.Set("token", ...)` → `jdb.X` → `response.ITEMS` / `response.HTTPError`), registered in `Routes`.

## Configuration
- `.env` — read via `et/envar`: server flags above plus `HOST`, `PATH_URL`, `RELSEG_SIZE`, `SYNC_ON_WRITE`, `TIMEZONE`, `MIN_THRESHOLD_COMPACT`, `TTL_TRANSACCION`, `DB_POOL`, `LANG`, and `REDIS_*`/`NATS_*` for `et`'s `cache`/`event`.
- `config.json` — a `nodes` list of cluster peers; not read by any Go code. Together with the unused `wal.go`, this is an intended but unbuilt clustering feature — nodes don't coordinate.
- Data lives under `DB_PATH_DATA` / `DB_PATH_WAL` / `DB_PATH_SYSTEM` (defaults under `./data/`).

## Framework: `github.com/cgalvisleon/et`
Everything outside storage and indexing comes from `et`: `server` (`Ettp`), `router` (chi-based), `response`, `claim` (JWT), `request` (context keys), `event` (pub/sub), `cache`, `jrpc`, `jrex` (JS triggers), `csv`/`xls`, `logs`, `envar`, `utility`, `reg` (ids), and `et` itself (`Json`, `Items`, `Condition`). Before writing a helper, check whether `et` already has it.

## Stale docs, don't rely on them
`README.md` and `AGENTS.md` describe an older architecture: a `cmd/client` REPL, TCP remote/embedded modes, SQL-text syntax (`CREATE TABLE`, `SELECT ... WHERE`), and packages `internal/catalog`, `internal/stmt`, `internal/jsql`, `internal/cli`, `internal/client`, `pkg/http`, `pkg/websocket`. **None of that exists anymore.** Trust this file and the source over those two.
