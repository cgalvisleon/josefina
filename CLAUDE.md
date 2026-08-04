# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Run server (single node)
```bash
gofmt -w . && go run ./cmd/server -port 1377 -http 3500
```

### Run multiple server nodes (cluster)
```bash
go run ./cmd/server -port 1377 -http 3500
go run ./cmd/server -port 1378 -http 3501
go run ./cmd/server -port 1379 -http 3502
```
Each instance is an independent node (`config.json`'s `nodes` list documents intended peer addresses but isn't wired up yet — see the `internal/jdb`/`internal/jsql` note below).

### Run client (REPL)
```bash
# Remote (TCP) mode — requires a running server
go run ./cmd/client -host 127.0.0.1:1377 -user <username> -password <password>

# Local (embedded) mode — boots jdb in-process, no server needed
go run ./cmd/client -user admin -password secret -database mydb
```

### Test
```bash
go test ./internal/stmt/...          # parser + dialect tests (fast, no server needed)
go test ./...                        # all packages
go test -run TestName ./internal/stmt/  # single test
```

### Build
```bash
go build ./cmd/server
go build ./cmd/client
```

### Format
```bash
gofmt -w .
```

### Go version
```bash
goenv local 1.25.0   # project uses Go 1.25.0 (see go.mod)
```

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

Josefina is a custom distributed document database engine written in Go with SQL-like query syntax. It has no external database dependency — all data is stored locally in its own file-based storage engine.

### Entry points
- `cmd/server/main.go` — starts the database server. Flags: `-port` (TCP, default 1377), `-http` (HTTP, default 3500), `-strict` (schema enforcement)
- `cmd/client/main.go` — starts an interactive REPL; omit `-host` for local embedded mode
- `cmd/store/main.go`, `cmd/test/main.go` — standalone scratch programs exercising `internal/store` and `internal/jdb` directly (not part of the public product surface)

### Layer breakdown

**`internal/store`** — Low-level WAL file storage engine
- `FileStore`: append-only segmented file store with an in-memory index (`map[string]*RecordRef`)
- Records are written to segment files (`segment-XXXXXX.dat`) and indexed by string key
- Tombstone-based deletions; automatic compaction when tombstones exceed 10% of index size
- Snapshots created on each segment roll-over
- Config: `RELSEG_SIZE` (segment size in MB, default 128), `SYNC_ON_WRITE` (default true)

**`internal/jdb`** — Schema, catalog, and database engine
- `DB` → `Schema` → `Model` → `Field` hierarchy. `DB`s are held in a package-level registry (`NewDb`, `LoadDb`, `GetDb` in `jdb.go`)
- Every `DB` auto-loads a set of core models on creation: `store`, `cache`, `users`, `sessions` (schema `""`) and `errors` (schema `.catalog`, the `sysSchema` constant in `db.go`) — these live inside the same database, not a separate global catalog DB
- `Model` owns a primary `FileStore` (key → document) plus one `FileStore` per secondary index and one in-memory B+ tree (`btree.go`, degree 32, `bpDegree`) per index — each `BTree` self-persists and reloads on `Model.Init()`
- `BTree` secondary indexes return `[]string` (primary keys) and support `Get`, `Insert`, `Delete`, `Equal`, `NotEqual`, `Between`, `NotBetween`, `More`, `MoreEq`, `Less`, `LessEq`, `Like`, `In`, `NotIn`, `Is`, `IsNot`, `Null`, `NotNull`, `ApplyCondition` (takes an `et.Condition` directly)
- JavaScript triggers (`Model` before/after insert/update/delete hooks, see `AddBeforeInsert` etc. and `fireTriggers` in `model.go`) run through `github.com/cgalvisleon/et/jrex` (`jrex.NewInstance()`), not an in-package VM wrapper
- Note: `internal/jsql/server.go` currently references `jdb.Node`, `jdb.NodeParams`, and `jdb.Load` for a clustered/TCP-owning node type — as of this writing that type does not exist in `internal/jdb` (only `DB`/`NewDb`/`LoadDb`/`GetDb` do), so that file will not compile until the node abstraction lands. Verify with `go build ./...` before relying on the jsql server layer

**`internal/stmt`** — Query language parser and SQL dialect translators
- Custom lexer (`lexer.go`) and per-statement parsers: `parser_ddl.go`, `parser_dml.go`, `parser_db.go`, `parser_user.go`, `parser_serie.go`, `parser_cache.go`, `parser_tx.go`, `parser_cmd.go`, `parser_json.go`, `parser_text.go`
- Statement types defined in `stmt_*.go` files; execution entry point is `ExecSQL(db, sql) (et.Items, error)` in `executor.go`
- Dialect subdirs: `internal/stmt/mysql`, `internal/stmt/oracle`, `internal/stmt/sqlserver` — each implement DDL/DML/TX translators and have their own tests
- SQL state can be switched per session: `SET SQL STATE MYSQL|POSTGRESQL|ORACLE|SQLSERVER|JOSEFINA`

**`internal/jsql`** — Public server/client/auth layer wrapping `internal/jdb` and `internal/stmt`
- `server.go`: singleton `Server` (embeds `*jdb.Node` — see the note above; this file is currently ahead of `internal/jdb`), created with `NewServer(port)`, mounts a `QueryService` (`query.go`) and starts the TCP transport
- `client.go`: TCP client (`github.com/cgalvisleon/et/tcp`) connecting to a running node, used by `cmd/client` in remote mode
- `auth.go`, `session.go`: authentication middleware, backed by `github.com/cgalvisleon/et/claim` (JWT)
- `dialect.go`: translates parsed `stmt.Stmt` values to a target SQL dialect (currently Postgres-flavored rendering; used together with `internal/stmt`'s `SET SQL STATE` dialects)
- `render.go`: REPL-facing output formatting (table printing, banners, colored messages) used by `cmd/client`

**`internal/server`** — Top-level server wiring (HTTP + WebSocket + TCP started from here)

**`internal/cli`** — Terminal rendering for the REPL (table formatting, output)

**`internal/client`** — REPL input loop and meta-commands (`\c`, `\timing`, `\i`, `\q`)

**`pkg/http`** — HTTP API layer using `go-chi/chi`

**`pkg/websocket`** — WebSocket hub using `gorilla/websocket`

**`internal/msg`** — Centralized error/message string constants (`MSG_*`), with English/Spanish variants selected by the `LANG` env var

### Configuration
- `config.json` — a `nodes` list of `host:port` cluster peer addresses; not currently read by any Go code (`cmd/server` takes `-port`/`-http`/`-strict` flags directly instead)
- `.env` — environment variables read via `envar`, including `LANG` (`en`/`es`, selects `internal/msg` message language), `RELSEG_SIZE`, `SYNC_ON_WRITE`, `TENNANT_NAME`, `TENNANT_PATH_DATA`, `TIMEZONE`, `MIN_THRESHOLD_COMPACT`, `TTL_TRANSACCION`, `HOST`, `PORT`, `RPC_PORT`, `TCP_PORT`, `PATH_URL`, `DEBUG`
- Data is persisted under `./data/<TENNANT_NAME>/dbs/<database>/`

### Key dependency
- `github.com/cgalvisleon/et` — shared utilities providing: `tcp` (transport), `ws` (websocket), `et` (JSON type/`Condition`), `claim` (JWT), `jrex` (JS trigger runtime), `logs`, `envar`, `utility`, `reg`, `strs`, `middleware`, `router`, `response`, `stdrout`, `timezone`

### Stale docs, don't rely on them
- `README.md` and `AGENTS.md` describe an older `internal/catalog` package and `pkg/sql` layer that no longer exist — that code was consolidated into `internal/jdb` and `internal/jsql`. Trust this file and the source over those two.
