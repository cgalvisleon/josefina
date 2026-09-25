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

**`internal/store`** — WAL storage engine (append-only binary key→value store). See the dedicated section below.

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

## `internal/store` — storage engine

A **single-node**, concurrency-safe, append-only (WAL-style) store of `id → []byte`. It is meant to be the per-node base for high availability, which will be a replication layer *above* it (likely in `internal/jdb`), not inside it. It stores raw bytes; JSON encoding is the caller's job. Doc comments in this package are in Spanish.

### Public API (this is the whole surface; everything else is private on purpose)
| Call | Semantics |
|---|---|
| `Open(pathData, pathWald, name, mode)` | Opens/creates. Segments in `pathData/segments/<name>/`; `pathWald/{snapshot,compact,recover}/<name>/`. `name` goes through `Normalize`. `ReadOnly`/`ReadWrite`. |
| `Close()`, `Empty()` | `Close` waits for compaction, fsyncs and retires all segments. `Empty` closes and deletes the data dir. |
| `Insert(id, data) (bool, error)` | Writes only if `id` is absent. |
| `Update(id, data) (bool, error)` | Writes only if `id` exists **and** the bytes differ (identical data → no log write, returns `false`). |
| `Delete(id) (bool, error)` | Writes a tombstone only if `id` exists. |
| `Get(id) ([]byte, bool, error)`, `IsExist`, `Count`, `Keys(asc, offset, limit)` | Reads. `limit <= 0` = no limit. |
| `ForEach(fn, asc, offset, limit)` | Disk reads run concurrently on `min(NumCPU, n)` goroutines, but `fn` is called **sequentially, in key order, on the caller's goroutine** — it needs no locking, and returning `false` stops exactly there. The record set is fixed at call start; no lock is held, so `fn` may call back into the store. |
| `Compact()` | Manual compaction (also automatic when tombstones > max(10% of keys, `MIN_THRESHOLD_COMPACT`)). |
| `WalSince(lsn)`, `ApplyWalEntry(e)`, `WalEntry` | Replication primitives (log shipping by LSN). Not wired to anything yet. |
| `Recover(pathData, pathWald, name) (et.Json, error)` | Offline repair; the store must be closed everywhere. Restores/cleans an interrupted compaction, drops the snapshot, and truncates each segment at its last valid record after copying the bad tail to `pathWald/recover/<name>/`. Returns a report. |
| `ToJson`, `ToString`, `IsDebug`, `Normalize` | Status/debug helpers; `Normalize` is also used by `jdb` for names. |

There is deliberately **no upsert**: callers compose it as `Update`, then `Insert` if the id did not exist (and `Update` again if that `Insert` lost a race).

### On-disk format
- Record: `[LSN:8][DataLen:4][CRC:4][IDLen:2][ID][Status:1][Data]`, big-endian; `Status` is `Active` or `Deleted` (tombstone, no data). **The CRC covers `Data` only**, not LSN/ID/status.
- Segments `segment-%06d.dat`, rotated at `RELSEG_SIZE` MB (default 128). A new snapshot is written on every rotation.
- Snapshot `state-<name>.snap` (version 2): index of all non-active segments plus the WAL counter, CRC-protected. Optional — if missing or corrupt, `Open` rebuilds the index by scanning every segment.
- Scans stop silently at the first torn/corrupt record (see `segment.scan`), which is also how `Recover` finds the valid end.

### Concurrency invariants (keep these when changing the package)
- `writeMu` serializes every mutation, and a mutation's existence check + log append + index update happen under **one** hold (`put`, `appendRecordLocked` require it held). Lock order is always `compactMu` → `writeMu` → `indexMu`.
- `indexMu` (RWMutex) guards `index`, `segments` and `active`.
- Readers **never hold `indexMu` during disk I/O**: they look up the ref and `acquire()` its segment under `RLock`, release the lock, read, then `release()`. Compaction and `Close` call `retire()` on old segments instead of closing them; the file closes when the last reader releases it. Never close a segment that might still have readers.
- Segment writes are synchronous (`os.File.Write`), so a ref published in the index is always readable. The first write error is sticky.
- Shared counters `WAL`, `TombStones` and `Size` are `counter[T]`; use `inc`/`dec`/`add`/`count`/`set`/`setMax`, never the fields. The index goes through `getIndex`/`setIndex`/`deleteIndex`/`countIndex`; `...Locked` variants are for callers already holding `indexMu`.
- `Compact` runs in 3 phases: (1) under `writeMu`, copy the index and record the cut point (segment + offset); (2) without locks, copy live records to a temp dir; (3) under `writeMu` + `indexMu`, replay everything written after the cut (Puts copied, Deletes written as tombstones), delete the snapshot, swap directories, retire the old segments, rebuild the snapshot. It keeps the newest tombstone so the WAL counter never regresses on rebuild.

### Known limitations (accepted, not bugs to "fix" unasked)
- `Open` does **not** run `Recover` automatically. After a crash with a torn tail, writes appended behind the garbage are lost on the next restart; after a crash between the two compaction renames, `Open` starts empty. Run `Recover` before `Open` after an unclean shutdown.
- One writer at a time; with `SYNC_ON_WRITE=true` (default) every write does an fsync, so write throughput is bounded by disk latency (no group commit).
- Compaction drops tombstones and overwritten records, so a replica behind the cut cannot catch up with `WalSince` (it would miss deletes) — the future HA layer needs log retention or full snapshot transfer.

### Testing the store
There are no committed tests. For ad-hoc tests: after `Open`, set `fs.MaxSegment` to a few KB to force rotation; use `t.Setenv("SYNC_ON_WRITE", "false")` for speed and `MIN_THRESHOLD_COMPACT` to control auto-compaction; always run with `-race`. Reopen the store at the end and compare against the in-memory state — that is how every change to this package has been verified.

## Patterns for extending

- **New server-level operation:** add a lowercase `(s *Server) jX(params et.Json) (et.Items, error)` in `internal/jdb/server.go` that validates the token and resolves the session, then a public wrapper `JX(params)` in `jdb.go` that checks `server != nil` and calls `server.Exec(params, server.jX)`. Never call the handler directly — `Server.Exec` pushes onto the request channel consumed by `cores*2+1` workers (or `DB_POOL`).
- **New JSON op inside an endpoint:** write a `func xQuery(db *DB, params []et.Json) ([]et.Json, error)` in `query.go` and add `{xQuery, params.ArrayJson("x")}` to the `queryJob` list in the matching `DB.jSystem`/`jQuery`/`jCommand` (`db.go`). Jobs run concurrently; the first error wins.
- **New catalog object for `define`/`describe`:** add a `case` in `defineQuery`/`describeQuery` that resolves the target DB with `db.server.getDb(define.Str("database"))` and delegates to a `DB.defineX`/`describeX` method.
- **New HTTP route:** handler in `pkg/server/router.go` follows the existing shape (bearer token → body → `body.Set("token", ...)` → `jdb.X` → `response.ITEMS` / `response.HTTPError`), registered in `Routes`.

## Configuration
- `.env` — read via `et/envar`: server flags above plus `HOST`, `PATH_URL`, `RELSEG_SIZE` (segment size in MB, default 128), `SYNC_ON_WRITE` (fsync per write, default true), `TIMEZONE`, `MIN_THRESHOLD_COMPACT` (default 1000), `TTL_TRANSACCION`, `DB_POOL`, `LANG`, and `REDIS_*`/`NATS_*` for `et`'s `cache`/`event`.
- `config.json` — a `nodes` list of cluster peers; not read by any Go code. Together with the unused `wal.go`, this is an intended but unbuilt clustering feature — nodes don't coordinate.
- Data lives under `DB_PATH_DATA` / `DB_PATH_WAL` / `DB_PATH_SYSTEM` (defaults under `./data/`).

## Framework: `github.com/cgalvisleon/et`
Everything outside storage and indexing comes from `et`: `server` (`Ettp`), `router` (chi-based), `response`, `claim` (JWT), `request` (context keys), `event` (pub/sub), `cache`, `jrpc`, `jrex` (JS triggers), `csv`/`xls`, `logs`, `envar`, `utility`, `reg` (ids), and `et` itself (`Json`, `Items`, `Condition`). Before writing a helper, check whether `et` already has it.

## Stale docs, don't rely on them
`README.md` and `AGENTS.md` describe an older architecture: a `cmd/client` REPL, TCP remote/embedded modes, SQL-text syntax (`CREATE TABLE`, `SELECT ... WHERE`), and packages `internal/catalog`, `internal/stmt`, `internal/jsql`, `internal/cli`, `internal/client`, `pkg/http`, `pkg/websocket`. **None of that exists anymore.** Trust this file and the source over those two.
